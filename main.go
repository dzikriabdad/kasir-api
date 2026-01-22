package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Produk struct {
	ID    int    `json:"id"`
	Nama  string `json:"nama"`
	Harga int    `json:"harga"`
	Stok  int    `json:"stok"`
}

var produk = []Produk{
	{ID: 1, Nama: "bebek bumbu hitam", Harga: 15000, Stok: 60},
	{ID: 2, Nama: "ayam bumbu htiam", Harga: 12000, Stok: 60},
	{ID: 3, Nama: "es teh", Harga: 3000, Stok: 120},
}

func getProdukByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/produk/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid produk id", http.StatusBadRequest)
		return
	}

	for _, p := range produk {
		if p.ID == id {
			json.NewEncoder(w).Encode(p)
			return
		}
	}
	http.Error(w, "produk belum ada", http.StatusNotFound)
}
func updateProduk(w http.ResponseWriter, r *http.Request) {
	//get id dari req
	idStr := strings.TrimPrefix(r.URL.Path, "/api/produk/")

	//ganti int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid produk id", http.StatusBadRequest)
		return
	}
	//get data dari req
	var updateProduk Produk
	err = json.NewDecoder(r.Body).Decode(&updateProduk)
	if err != nil {
		http.Error(w, "BAD REQUEST", http.StatusBadRequest)

		return
	}

	//loop pro, cari id, ganti sesuai data dari req
	for i := range produk {
		if produk[i].ID == id {
			updateProduk.ID = id
			produk[i] = updateProduk
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(updateProduk)
			return
		}
		http.Error(w, "produk belum ada", http.StatusNotFound)
	}

}
func deleteProduk(w http.ResponseWriter, r *http.Request) {
	//get id
	idStr := strings.TrimPrefix(r.URL.Path, "/api/produk/")
	//ganti id to int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid produk id", http.StatusBadRequest)
		return
	}
	//loop cari ID, dapat index yg akan dihapus
	for i, p := range produk {
		if p.ID == id {
			//make slice baru berdasarkan data sebelum dan sesudah index
			produk = append(produk[:i], produk[i+1:]...)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"massage": "sukses hapus",
			})
			return
		}
	}
	http.Error(w, "produk belum ada", http.StatusNotFound)
}

func main() {

	//GET localhost:8080/api/produk/{id}
	//PUT localhost:8080/api/produk/{id}
	//DELETE localhost:8080/api/produk/{id}
	http.HandleFunc("/api/produk/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			getProdukByID(w, r)
		} else if r.Method == "PUT" {
			updateProduk(w, r)
		} else if r.Method == "DELETE" {
			deleteProduk(w, r)
		}

	})

	// GET localhost:8080/api/produk
	// POST localhost:8080/api/produk
	http.HandleFunc("/api/produk", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-type", "application/json")
			json.NewEncoder(w).Encode(produk)
		} else if r.Method == "POST" {

			//baca data dari req
			var produkbaru Produk
			err := json.NewDecoder(r.Body).Decode(&produkbaru)
			if err != nil {
				http.Error(w, "BAD REQUEST", http.StatusBadRequest)

				return
			}
			//masukin data ke var produk
			produkbaru.ID = len(produk) + 1
			produk = append(produk, produkbaru)

			w.Header().Set("Content-type", "application/json")
			w.WriteHeader(http.StatusCreated) //201
			json.NewEncoder(w).Encode(produkbaru)
		}

	})
	// localhost:8080/health
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"masage": "API RUNGING",
		})
		w.Write([]byte("ok"))
	})
	fmt.Println("SERVER RUNING DI localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("GAGAL RUNING SERVER")
	}
}
