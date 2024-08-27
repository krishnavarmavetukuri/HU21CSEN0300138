package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type KV_Product struct {
	ID           string  `json:"id"`
	ProductName  string  `json:"productName"`
	Price        float64 `json:"price"`
	Rating       float64 `json:"rating"`
	Discount     int     `json:"discount"`
	Availability string  `json:"availability"`
	Company      string  `json:"company"`
}

type KV_APIResponse []KV_Product

var KV_productsMap = make(map[string]KV_Product)
var KV_baseAPI = "http://20.244.56.144/test/companies"

func main() {
	http.HandleFunc("/categories/", KV_handleCategories)

	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// KV_handleCategories handles requests for both fetching products and fetching product details.
func KV_handleCategories(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/categories/")
	pathParts := strings.Split(path, "/")

	if len(pathParts) == 2 && pathParts[1] == "products" {
		KV_GetTopProducts(w, r, pathParts[0])
	} else if len(pathParts) == 3 && pathParts[1] == "products" {
		KV_GetProductByID(w, r, pathParts[0], pathParts[2])
	} else {
		http.Error(w, "Invalid request path", http.StatusBadRequest)
	}
}

// KV_GetTopProducts handles the GET request to fetch top 'n' products in a specified category.
func KV_GetTopProducts(w http.ResponseWriter, r *http.Request, KV_category string) {
	KV_topN, _ := strconv.Atoi(r.URL.Query().Get("n"))
	if KV_topN == 0 {
		KV_topN = 10 // default to top 10
	}
	KV_page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if KV_page == 0 {
		KV_page = 1
	}

	KV_sortBy := r.URL.Query().Get("sort")
	KV_order := r.URL.Query().Get("order")

	KV_products := KV_fetchProductsFromAllCompanies(KV_category, KV_topN, KV_sortBy, KV_order)

	KV_startIndex := (KV_page - 1) * KV_topN
	KV_endIndex := KV_startIndex + KV_topN
	if KV_startIndex > len(KV_products) {
		KV_startIndex = len(KV_products)
	}
	if KV_endIndex > len(KV_products) {
		KV_endIndex = len(KV_products)
	}

	KV_products = KV_products[KV_startIndex:KV_endIndex]

	for i := range KV_products {
		KV_id := KV_generateUniqueID()
		KV_products[i].ID = KV_id
		KV_productsMap[KV_id] = KV_products[i]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(KV_products)
}

// KV_GetProductByID handles the GET request to fetch details of a specific product by its ID.
func KV_GetProductByID(w http.ResponseWriter, r *http.Request, KV_category, KV_productID string) {
	KV_product, found := KV_productsMap[KV_productID]
	if !found {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(KV_product)
}

// KV_fetchProductsFromAllCompanies fetches the top products from all companies for the given category.
func KV_fetchProductsFromAllCompanies(KV_category string, KV_topN int, KV_sortBy, KV_order string) []KV_Product {
	KV_companies := []string{"AMZ", "FLP", "SNP", "MYN", "AZO"}
	var KV_wg sync.WaitGroup
	KV_productChannel := make(chan KV_Product)

	for _, KV_company := range KV_companies {
		KV_wg.Add(1)
		go func(KV_company string) {
			defer KV_wg.Done()
			KV_products := KV_fetchProductsFromCompany(KV_company, KV_category, KV_topN)
			for _, KV_product := range KV_products {
				KV_product.Company = KV_company
				KV_productChannel <- KV_product
			}
		}(KV_company)
	}

	go func() {
		KV_wg.Wait()
		close(KV_productChannel)
	}()

	var KV_products []KV_Product
	for KV_product := range KV_productChannel {
		KV_products = append(KV_products, KV_product)
	}

	KV_sortProducts(KV_products, KV_sortBy, KV_order)

	return KV_products
}

// KV_fetchProductsFromCompany fetches products from a single company.
type KV_CompanyResponse struct {
	Products KV_APIResponse `json:"products"`
}

func KV_fetchProductsFromCompany(KV_company, KV_category string, KV_topN int) []KV_Product {
	KV_url := fmt.Sprintf("%s/%s/categories/%s/products?top=%d&minPrice=1&maxPrice=10000", KV_baseAPI, KV_company, KV_category, KV_topN)
	resp, err := http.Get(KV_url)
	if err != nil {
		log.Printf("Failed to fetch products from %s: %v", KV_company, err)
		return nil
	}
	defer resp.Body.Close()

	var KV_companyResp KV_CompanyResponse
	if err := json.NewDecoder(resp.Body).Decode(&KV_companyResp); err != nil {
		log.Printf("Failed to decode response from %s: %v", KV_company, err)
		return nil
	}

	return KV_companyResp.Products
}

// KV_sortProducts sorts the products based on the given field and order.
func KV_sortProducts(KV_products []KV_Product, KV_sortBy, KV_order string) {
	sort.Slice(KV_products, func(i, j int) bool {
		switch KV_sortBy {
		case "price":
			if KV_order == "desc" {
				return KV_products[i].Price > KV_products[j].Price
			}
			return KV_products[i].Price < KV_products[j].Price
		case "rating":
			if KV_order == "desc" {
				return KV_products[i].Rating > KV_products[j].Rating
			}
			return KV_products[i].Rating < KV_products[j].Rating
		case "discount":
			if KV_order == "desc" {
				return KV_products[i].Discount > KV_products[j].Discount
			}
			return KV_products[i].Discount < KV_products[j].Discount
		case "company":
			if KV_order == "desc" {
				return strings.Compare(KV_products[i].Company, KV_products[j].Company) > 0
			}
			return strings.Compare(KV_products[i].Company, KV_products[j].Company) < 0
		default:
			return KV_products[i].Rating > KV_products[j].Rating
		}
	})
}

// KV_generateUniqueID generates a unique ID using a simple combination of random numbers and time.
func KV_generateUniqueID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%d-%d", rand.Intn(100000), time.Now().UnixNano())
}
