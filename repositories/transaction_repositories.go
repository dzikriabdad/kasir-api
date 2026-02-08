package repositories

import (
	"database/sql"
	"fmt"
	"kasir-api/models"
	"strings"
	"time"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (repo *TransactionRepository) CreateTransaction(items []models.CheckoutItem) (*models.Transaction, error) {
	// 1. Memulai sesi transaksi database (Atomic operation)
	tx, err := repo.db.Begin()
	if err != nil {
		return nil, err
	}

	// Memastikan transaksi dibatalkan (rollback) jika terjadi error di tengah jalan
	// Jika tx.Commit() dipanggil di akhir, defer ini tidak akan melakukan apa-apa
	defer tx.Rollback()

	totalAmount := 0
	details := make([]models.TransactionDetail, 0)

	// 2. Loop setiap item yang dibeli untuk validasi stok dan harga
	for _, item := range items {
		var productPrice, stock int
		var productName string

		// Ambil data produk terbaru dari database untuk memastikan harga dan stok valid
		err := tx.QueryRow("SELECT name, price, stock FROM products WHERE id = $1", item.ProductID).Scan(&productName, &productPrice, &stock)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product id %d not found", item.ProductID)
		}
		if err != nil {
			return nil, err
		}

		// (Opsional) Kamu bisa tambah cek stok di sini: if stock < item.Quantity { ... }

		// Hitung subtotal per item
		subtotal := productPrice * item.Quantity
		totalAmount += subtotal

		// 3. Kurangi stok produk di database
		_, err = tx.Exec("UPDATE products SET stock = stock - $1 WHERE id = $2", item.Quantity, item.ProductID)
		if err != nil {
			return nil, err
		}

		// Masukkan data ke slice 'details' untuk disimpan nanti
		details = append(details, models.TransactionDetail{
			ProductID:   item.ProductID,
			ProductName: productName,
			Quantity:    item.Quantity,
			Subtotal:    subtotal,
		})
	}

	// 4. Masukkan data utama transaksi ke tabel 'transactions'
	// Menggunakan RETURNING id untuk mendapatkan ID transaksi yang baru saja dibuat
	var transactionID int
	err = tx.QueryRow("INSERT INTO transactions (total_amount) VALUES ($1) RETURNING id", totalAmount).Scan(&transactionID)
	if err != nil {
		return nil, err
	}

	if len(details) > 0 {
		valueStrings := make([]string, 0, len(details))
		valueArgs := make([]interface{}, 0, len(details)*4)

		for i, d := range details {
			// Update struct untuk return value nanti
			details[i].TransactionID = transactionID

			// Siapkan placeholder ($1, $2, $3, $4), ($5, $6...), dst
			n := i * 4
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4))

			valueArgs = append(valueArgs, transactionID, d.ProductID, d.Quantity, d.Subtotal)
		}

		stmt := fmt.Sprintf("INSERT INTO transaction_details (transaction_id, product_id, quantity, subtotal) VALUES %s",
			strings.Join(valueStrings, ","))

		_, err = tx.Exec(stmt, valueArgs...)
		if err != nil {
			return nil, err
		}
	}
	// 6. Jika semua proses lancar, simpan semua perubahan secara permanen (Commit)
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Mengembalikan object Transaction yang sudah lengkap dengan ID dan rinciannya
	return &models.Transaction{
		ID:          transactionID,
		TotalAmount: totalAmount,
		Details:     details,
	}, nil
}
func (repo *TransactionRepository) GetReportByRange(startTime, endTime time.Time) (*models.ReportToday, error) {
	var report models.ReportToday

	querySummary := `
		SELECT 
			COALESCE(SUM(total_amount), 0), 
			COUNT(id) 
		FROM transactions 
		WHERE created_at >= $1 AND created_at < $2
	`
	err := repo.db.QueryRow(querySummary, startTime, endTime).Scan(&report.TotalRevenue, &report.TotalTransaksi)
	if err != nil {
		return nil, err
	}

	queryBestSelling := `
		SELECT 
			p.name, 
			COALESCE(SUM(td.quantity), 0) as total_qty
		FROM transaction_details td
		JOIN products p ON td.product_id = p.id
		JOIN transactions t ON td.transaction_id = t.id
		WHERE t.created_at >= $1 AND t.created_at < $2
		GROUP BY p.name
		ORDER BY total_qty DESC
		LIMIT 1
	`
	err = repo.db.QueryRow(queryBestSelling, startTime, endTime).Scan(&report.ProdukTerlaris.Name, &report.ProdukTerlaris.QtyTerjual)
	if err == sql.ErrNoRows {

		report.ProdukTerlaris = models.BestSelling{Name: "-", QtyTerjual: 0}
	} else if err != nil {
		return nil, err
	}

	return &report, nil
}
