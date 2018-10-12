package shoppinglist

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"

	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

var listofItems []Item
var totalP float64

//Add a new item to the list x addItem
//Remove a single item x removeItem
//Remove all items x removeAllItems
//Return the total price for all your items x totalPrice
//Return all items that you need to buy in a single supermarket x showItems

//Item  is a representation of a peritem son
type Item struct {
	Name        string  `json:"name"`
	Supermarket string  `json:"supermarket"`
	Price       float64 `json:"price"`
	ImageURL    string  `json:"imageURL"`
}

//show items on a list
func showItems(w http.ResponseWriter, r *http.Request) {

	supermarketName := r.FormValue("Supermarket")

	//til að taka á case-sensitive í supermarket
	if supermarketName == "" {
		supermarketName = r.FormValue("supermarket")
	}

	var returnItems []Item

	// set the Content-Type header.
	w.Header().Set("Content-Type", "application/json")

	ctx := appengine.NewContext(r)

	// create a new query on the kind Item
	q := datastore.NewQuery("Item")

	//ætla bara að nota filterinn ef supermarket er ekki tómt
	if supermarketName != "" {
		q = q.Filter("Supermarket =", supermarketName)
	}
	// select only values where field Age is 10 or lower

	q = q.Order("Supermarket").
		Order("Name").
		Order("Price")

	// and finally execute the query retrieving all values into returnItems.
	_, err := q.GetAll(ctx, &returnItems)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//encode p to the output.
	enc := json.NewEncoder(w)
	encodeErr := enc.Encode(returnItems)
	if encodeErr != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//tek þetta út af því að ég vil fá þetta sem json object, þess vegna er ég með encoding
	//fmt.Fprintln(w, returnItems)
}

//add a single item
func addItem(w http.ResponseWriter, r *http.Request) {

	var item Item

	dec := json.NewDecoder(r.Body)

	//&item is a pointer to the variable item
	err := dec.Decode(&item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// create a new App Engine context from the HTTP request.
	ctx := appengine.NewContext(r)

	// create a new incomplete key of kind Item.
	key := datastore.NewIncompleteKey(ctx, "Item", nil)

	// put i in the datastore.
	key, erro := datastore.Put(ctx, key, &item)
	if erro != nil {
		http.Error(w, erro.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "item stored with key %v", key)

}

//add a single item
func addItemFromForm(w http.ResponseWriter, r *http.Request) {

	imageURL, err := uploadFileFromForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
		//return nil, fmt.Errorf("could not upload file: %v", err)
	}
	//ef formið inniheldur imageURL þá nota ég það ef engin mynd er sett í formið
	if imageURL == "" {
		imageURL = r.FormValue("imageURL")
		http.Error(w, "dummy", http.StatusInternalServerError)
		return
	}
	//convert to float64
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)

	item := Item{
		Name:        r.FormValue("name"),
		Supermarket: r.FormValue("supermarket"),
		Price:       price,
		ImageURL:    imageURL,
	}

	// create a new App Engine context from the HTTP request.
	ctx := appengine.NewContext(r)

	// create a new incomplete key of kind Item.
	key := datastore.NewIncompleteKey(ctx, "Item", nil)

	// put i in the datastore.
	key, erro := datastore.Put(ctx, key, &item)
	if erro != nil {
		http.Error(w, "erro2", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "item stored with key %v", key)

}

//remove all items
func removeAllItems(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	var itemList []Item

	// create a new query on the kind Person
	q := datastore.NewQuery("Item")

	// and finally execute the query retrieving all values into itemList.
	keys, err := q.GetAll(ctx, &itemList)
	err = datastore.DeleteMulti(ctx, keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

}

//show total price of list
func totalPrice(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	var itemList []Item
	var sum float64

	// create a new query on the kind Person
	q := datastore.NewQuery("Item")

	// and finally execute the query retrieving all values into itemlist.
	_, err := q.GetAll(ctx, &itemList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for index := range itemList {
		item := itemList[index]
		sum += item.Price
	}
	//encode the list to the output.
	enc := json.NewEncoder(w)
	encodeErr := enc.Encode(sum)
	if encodeErr != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//fmt.Fprintln(w, sum)
}

//remove single item
func removeItem(w http.ResponseWriter, r *http.Request) {
	var removeItem Item
	dec := json.NewDecoder(r.Body)

	//&item is a pointer to the variable item
	err := dec.Decode(&removeItem)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// create a new App Engine context from the HTTP request.
	ctx := appengine.NewContext(r)
	q := datastore.NewQuery("Item")

	q = q.Filter("Supermarket =", removeItem.Supermarket).
		Filter("Name =", removeItem.Name).
		Filter("Price =", removeItem.Price)

		// ath þarf að sækja key hérna
	q = q.KeysOnly().Limit(1)

	keys, queryErr := q.GetAll(ctx, nil)
	if queryErr != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for index := range keys {
		datastore.Delete(ctx, keys[index])
	}

}

// uploadFileFromForm uploads a file if it's present in the "image" form field.
func uploadFileFromForm(r *http.Request) (url string, err error) {
	f, fh, err := r.FormFile("image")
	if err == http.ErrMissingFile {
		return "", errors.New("villa0")
	}
	if err != nil {
		return "", err
	}
	//var err error
	var StorageBucket *storage.BucketHandle

	StorageBucketName := "shoppinglist-218417.appspot.com"
	ctx := appengine.NewContext(r)
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	StorageBucket = client.Bucket(StorageBucketName)
	//StorageBucket, err = configureStorage(StorageBucketName)

	if err != nil {
		log.Fatal(err)
		log.Fatal(errors.New("Svenni 1"))
		return "", errors.New("villa1")
	}

	if StorageBucket == nil {
		return "", errors.New("storage bucket is missing - check config.go")
	}

	// random filename, retaining existing extension.
	//name := "yadablada" + path.Ext(fh.Filename) //uuid.Must(uuid.NewV4()).String() + path.Ext(fh.Filename)

	name := uuid.Must(uuid.NewV4()).String() + path.Ext(fh.Filename)
	ctx = appengine.NewContext(r)
	w := StorageBucket.Object(name).NewWriter(ctx)
	w.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
	w.ContentType = fh.Header.Get("Content-Type")

	// Entries are immutable, be aggressive about caching (1 day).
	w.CacheControl = "public, max-age=86400"

	if _, err := io.Copy(w, f); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	const publicURL = "https://storage.googleapis.com/%s/%s"
	return fmt.Sprintf(publicURL, StorageBucketName, name), nil
}

func init() {
	r := mux.NewRouter()

	r.HandleFunc("/items", showItems).Methods("GET")

	r.HandleFunc("/items", addItemFromForm).Methods("POST")

	r.HandleFunc("/removeAll", removeAllItems).Methods("POST")

	r.HandleFunc("/sum", totalPrice).Methods("GET")

	r.HandleFunc("/removeOne", removeItem).Methods("POST")

	// handle all requests with the Gorilla router.
	http.Handle("/", r)

}
