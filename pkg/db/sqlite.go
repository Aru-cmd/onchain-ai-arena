package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// DB is SQLite supervisor for fake testnet portfolios.
// Master (orchestrator) owns DB, gives each bot initial fake USD.
type DB struct {
	sql *sql.DB
	mu  sync.Mutex
	path string
}

// Open opens or creates SQLite DB at path. Use ":memory:" for tests.
func Open(path string) (*DB, error) {
	if path != "" && path != ":memory:" {
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
	}
	dsn := path
	if path != ":memory:" {
		// modernc.org/sqlite dsn with WAL for concurrency
		dsn = fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)
	}
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1) // SQLite single writer
	db := &DB{sql: sqlDB, path: path}
	if err := db.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func (d *DB) migrate() error {
	_, err := d.sql.Exec(`
	CREATE TABLE IF NOT EXISTS portfolios (
		agent_id TEXT PRIMARY KEY,
		usd REAL NOT NULL DEFAULT 0,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS holdings (
		agent_id TEXT NOT NULL,
		token TEXT NOT NULL,
		amount REAL NOT NULL DEFAULT 0,
		avg_price REAL NOT NULL DEFAULT 0,
		PRIMARY KEY (agent_id, token),
		FOREIGN KEY (agent_id) REFERENCES portfolios(agent_id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS trades (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_id TEXT NOT NULL,
		token TEXT NOT NULL,
		action TEXT NOT NULL,
		amount_usd REAL,
		amount_token REAL,
		price REAL,
		tx_hash TEXT,
		reason TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`)
	return err
}

func (d *DB) Close() error {
	if d.sql != nil {
		return d.sql.Close()
	}
	return nil
}

// EnsureAgent creates portfolio with initial USD if not exists.
func (d *DB) EnsureAgent(agentID string, initialUSD float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.sql.Exec(`INSERT OR IGNORE INTO portfolios(agent_id, usd) VALUES (?, ?)`, agentID, initialUSD)
	return err
}

// GetPortfolio returns USD + holdings for agent.
func (d *DB) GetPortfolio(agentID string) (usd float64, holdings map[string]float64, avg map[string]float64, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	err = d.sql.QueryRow(`SELECT usd FROM portfolios WHERE agent_id=?`, agentID).Scan(&usd)
	if err != nil {
		return 0, nil, nil, err
	}
	rows, err := d.sql.Query(`SELECT token, amount, avg_price FROM holdings WHERE agent_id=?`, agentID)
	if err != nil {
		return usd, nil, nil, err
	}
	defer rows.Close()
	holdings = make(map[string]float64)
	avg = make(map[string]float64)
	for rows.Next() {
		var token string
		var amt, avgP float64
		if err := rows.Scan(&token, &amt, &avgP); err != nil {
			return usd, holdings, avg, err
		}
		holdings[token] = amt
		avg[token] = avgP
	}
	return usd, holdings, avg, rows.Err()
}

// Buy updates DB: deduct USD, add holdings, update avg_price. Called by orchestrator only.
func (d *DB) Buy(agentID, token string, amountUSD, price float64, txHash, reason string) error {
	if price <= 0 {
		return fmt.Errorf("invalid price")
	}
	amountToken := amountUSD / price
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var usd float64
	if err := tx.QueryRow(`SELECT usd FROM portfolios WHERE agent_id=?`, agentID).Scan(&usd); err != nil {
		return err
	}
	if usd < amountUSD {
		return fmt.Errorf("insufficient fake USD: have %.2f need %.2f", usd, amountUSD)
	}
	var oldAmt, oldAvg float64
	err = tx.QueryRow(`SELECT amount, avg_price FROM holdings WHERE agent_id=? AND token=?`, agentID, token).Scan(&oldAmt, &oldAvg)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	newAmt := oldAmt + amountToken
	newAvg := price
	if oldAmt > 0 {
		newAvg = (oldAmt*oldAvg + amountToken*price) / newAmt
	}
	_, err = tx.Exec(`INSERT INTO holdings(agent_id, token, amount, avg_price) VALUES (?,?,?,?)
		ON CONFLICT(agent_id, token) DO UPDATE SET amount=?, avg_price=?`, agentID, token, newAmt, newAvg, newAmt, newAvg)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE portfolios SET usd = usd - ?, updated_at=CURRENT_TIMESTAMP WHERE agent_id=?`, amountUSD, agentID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO trades(agent_id, token, action, amount_usd, amount_token, price, tx_hash, reason) VALUES (?,?,?,?,?,?,?,?)`,
		agentID, token, "BUY", amountUSD, amountToken, price, txHash, reason)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Sell updates DB: add USD, deduct holdings.
func (d *DB) Sell(agentID, token string, amountToken, price float64, txHash, reason string) error {
	if price <= 0 {
		return fmt.Errorf("invalid price")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldAmt float64
	if err := tx.QueryRow(`SELECT amount FROM holdings WHERE agent_id=? AND token=?`, agentID, token).Scan(&oldAmt); err != nil {
		return fmt.Errorf("no holdings for %s: %w", token, err)
	}
	if oldAmt < amountToken {
		return fmt.Errorf("insufficient holdings: have %.6f need %.6f", oldAmt, amountToken)
	}
	newAmt := oldAmt - amountToken
	if newAmt < 1e-9 {
		_, err = tx.Exec(`DELETE FROM holdings WHERE agent_id=? AND token=?`, agentID, token)
	} else {
		_, err = tx.Exec(`UPDATE holdings SET amount=? WHERE agent_id=? AND token=?`, newAmt, agentID, token)
	}
	if err != nil {
		return err
	}
	proceeds := amountToken * price
	_, err = tx.Exec(`UPDATE portfolios SET usd = usd + ?, updated_at=CURRENT_TIMESTAMP WHERE agent_id=?`, proceeds, agentID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO trades(agent_id, token, action, amount_usd, amount_token, price, tx_hash, reason) VALUES (?,?,?,?,?,?,?,?)`,
		agentID, token, "SELL", proceeds, amountToken, price, txHash, reason)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Leaderboard returns total value per agent using given prices.
func (d *DB) Leaderboard(prices map[string]float64) (map[string]float64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rows, err := d.sql.Query(`SELECT agent_id, usd FROM portfolios`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]float64)
	agents := []struct{ id string; usd float64 }{}
	for rows.Next() {
		var id string
		var usd float64
		if err := rows.Scan(&id, &usd); err != nil {
			return nil, err
		}
		agents = append(agents, struct{ id string; usd float64 }{id, usd})
	}
	for _, a := range agents {
		total := a.usd
		hRows, err := d.sql.Query(`SELECT token, amount FROM holdings WHERE agent_id=?`, a.id)
		if err != nil {
			return nil, err
		}
		for hRows.Next() {
			var token string
			var amt float64
			_ = hRows.Scan(&token, &amt)
			if p, ok := prices[token]; ok {
				total += amt * p
			}
		}
		hRows.Close()
		result[a.id] = total
	}
	return result, nil
}

// Path returns db path.
func (d *DB) Path() string { return d.path }
