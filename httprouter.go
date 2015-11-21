package main

import (
	"assignment3/permutation"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
)

var id int
var hmap map[int]DataStorage

type Request struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
}

type Response struct {
	Id         bson.ObjectId `json:"id" bson:"_id"`
	Name       string        `json:"name" bson:"name"`
	Address    string        `json:"address" bson:"address"`
	City       string        `json:"city" bson:"city"`
	State      string        `json:"state" bson:"state"`
	Zip        string        `json:"zip" bson:"zip"`
	Coordinate struct {
		Lat float64 `json:"lat" bson:"lat"`
		Lng float64 `json:"lng" bson:"lng"`
	} `json:"coordinate" bson:"coordinate"`
}

type MyJsonName struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
			Viewport     struct {
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"viewport"`
		} `json:"geometry"`
		PlaceID string   `json:"place_id"`
		Types   []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}

type UberAPIResponse struct {
	Prices []struct {
		CurrencyCode    string  `json:"currency_code"`
		DisplayName     string  `json:"display_name"`
		Distance        float64 `json:"distance"`
		Duration        int     `json:"duration"`
		Estimate        string  `json:"estimate"`
		HighEstimate    int     `json:"high_estimate"`
		LowEstimate     int     `json:"low_estimate"`
		ProductID       string  `json:"product_id"`
		SurgeMultiplier int     `json:"surge_multiplier"`
	} `json:"prices"`
}

type TripRequest struct {
	Starting_from_location_id bson.ObjectId
	Location_ids              []bson.ObjectId
}

type TripResponse struct {
	Id                        int
	Status                    string
	Starting_from_location_id bson.ObjectId
	Best_route_location_id    []bson.ObjectId
	Total_uber_costs          int
	Total_uber_duration       int
	Total_distance            float64
}

type DataStorage struct {
	Id int
	// Product_id                string
	Index                     int
	Status                    string
	Starting_from_location_id bson.ObjectId
	Best_route_location_id    []bson.ObjectId
	Total_uber_costs          int
	Total_uber_duration       int
	Total_distance            float64
}

type UberRequestResponse struct {
	RequestID       string  `json:"request_id"`
	Status          string  `json:"status"`
	Vehicle         string  `json:"vehicle"`
	Driver          string  `json:"driver"`
	Location        string  `json:"location"`
	ETA             int     `json:"eta"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
}

type CarResponse struct {
	Id                           int
	Status                       string
	Starting_from_location_id    bson.ObjectId
	Next_destination_location_id bson.ObjectId
	Best_route_location_id       []bson.ObjectId
	Total_uber_costs             int
	Total_uber_duration          int
	Total_distance               float64
	Uber_wait_time_eta           int
}

type UserRequest struct {
	Product_id      string  `json:"product_id"`
	Start_latitude  float64 `json:"start_latitude"`
	Start_longitude float64 `json:"start_longitude"`
	End_latitude    float64 `json:"end_latitude"`
	End_longitude   float64 `json:"end_longitude"`
}

type UberResponse struct {
	Driver          interface{} `json:"driver"`
	Eta             int         `json:"eta"`
	Location        interface{} `json:"location"`
	RequestID       string      `json:"request_id"`
	Status          string      `json:"status"`
	SurgeMultiplier float64     `json:"surge_multiplier"`
	Vehicle         interface{} `json:"vehicle"`
}

var session *mgo.Session

func Create(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	var userInfo Request
	json.NewDecoder(req.Body).Decode(&userInfo)
	response := Response{}
	response.Id = bson.NewObjectId()
	response.Name = userInfo.Name
	response.Address = userInfo.Address
	response.City = userInfo.City
	response.State = userInfo.State
	response.Zip = userInfo.Zip
	requestURL := createURL(response.Address, response.City, response.State)
	getLocation(&response, requestURL)
	session.DB("cmpe273").C("assignment2").Insert(response)

	location, _ := json.Marshal(response)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(201)
	fmt.Fprintf(rw, "%s", location)
}

func CreateTrip(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	var tripReq TripRequest
	// var coordinateData [4]float64
	json.NewDecoder(req.Body).Decode(&tripReq)
	// fmt.Println(tripReq)
	// fmt.Println(len(tripReq.Location_ids))

	dataStorage := DataStorage{}
	tripRes := TripResponse{}
	tripRes.Id = getID()
	tripRes.Status = "planning"
	tripRes.Starting_from_location_id = tripReq.Starting_from_location_id
	tripRes.Best_route_location_id = tripReq.Location_ids

	getBestRoute(&tripRes, &dataStorage, tripReq.Starting_from_location_id, tripReq.Location_ids)
	//bestRouteLocation(&tripRes, &dataStorage, tripReq.Starting_from_location_id, tripReq.Location_ids)

	hmap[tripRes.Id] = dataStorage

	trip, _ := json.Marshal(tripRes)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(201)
	fmt.Fprintf(rw, "%s", trip)
}

func Get(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}
	userId := bson.ObjectIdHex(id)
	response := Response{}
	if err := session.DB("cmpe273").C("assignment2").FindId(userId).One(&response); err != nil {
		w.WriteHeader(404)
		return
	}
	location, _ := json.Marshal(response)
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", location)
}

func GetTrip(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	tId := p.ByName("trip_id")
	checkID, _ := strconv.Atoi(tId)
	var dataStorage DataStorage
	findTarget := false

	for key, value := range hmap {
		if key == checkID {
			dataStorage = value
			findTarget = true
		}
	}

	if findTarget == false {
		w.WriteHeader(404)
		return
	}

	tripRes := TripResponse{}
	tripRes.Id = dataStorage.Id
	tripRes.Status = dataStorage.Status
	tripRes.Starting_from_location_id = dataStorage.Starting_from_location_id
	tripRes.Best_route_location_id = dataStorage.Best_route_location_id
	tripRes.Total_uber_costs = dataStorage.Total_uber_costs
	tripRes.Total_distance = dataStorage.Total_distance
	tripRes.Total_uber_duration = dataStorage.Total_uber_duration

	trip, _ := json.Marshal(tripRes)
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", trip)
}

func Update(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}
	var userInfo Request
	json.NewDecoder(r.Body).Decode(&userInfo)
	response := Response{}
	response.Id = bson.ObjectIdHex(id)
	response.Name = userInfo.Name
	response.Address = userInfo.Address
	response.City = userInfo.City
	response.State = userInfo.State
	response.Zip = userInfo.Zip
	requestURL := createURL(response.Address, response.City, response.State)
	getLocation(&response, requestURL)

	if err := session.DB("cmpe273").C("assignment2").Update(bson.M{"_id": response.Id}, bson.M{"$set": bson.M{"address": response.Address, "city": response.City, "state": response.State, "zip": response.Zip, "coordinate.lat": response.Coordinate.Lat, "coordinate.lng": response.Coordinate.Lng}}); err != nil {
		w.WriteHeader(404)
		return
	}

	if err := session.DB("cmpe273").C("assignment2").FindId(response.Id).One(&response); err != nil {
		w.WriteHeader(404)
		return
	}

	location, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", location)
}

func CarRequest(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	tId := p.ByName("trip_id")
	checkID, _ := strconv.Atoi(tId)
	var dataStorage DataStorage
	findTarget := false

	for key, value := range hmap {
		if key == checkID {
			dataStorage = value
			findTarget = true
		}
	}

	if findTarget == false {
		w.WriteHeader(404)
		return
	}

	var startLat float64
	var startLng float64
	var endLat float64
	var endLng float64
	carRes := CarResponse{}
	response := Response{}

	if dataStorage.Index == 0 {
		if err := session.DB("cmpe273").C("assignment2").FindId(dataStorage.Starting_from_location_id).One(&response); err != nil {
			// w.WriteHeader(404)
			return
		}
		startLat = response.Coordinate.Lat
		startLng = response.Coordinate.Lng

		if err := session.DB("cmpe273").C("assignment2").FindId(dataStorage.Best_route_location_id[0]).One(&response); err != nil {
			// w.WriteHeader(404)
			return
		}
		endLat = response.Coordinate.Lat
		endLng = response.Coordinate.Lng
		uberAPI(&carRes, dataStorage, startLat, startLng, endLat, endLng)
		carRes.Status = "requesting"
		carRes.Starting_from_location_id = dataStorage.Starting_from_location_id
		carRes.Next_destination_location_id = dataStorage.Best_route_location_id[0]
	} else if dataStorage.Index == len(dataStorage.Best_route_location_id) {
		if err := session.DB("cmpe273").C("assignment2").FindId(dataStorage.Best_route_location_id[len(dataStorage.Best_route_location_id)-1]).One(&response); err != nil {
			// w.WriteHeader(404)
			return
		}
		startLat = response.Coordinate.Lat
		startLng = response.Coordinate.Lng

		if err := session.DB("cmpe273").C("assignment2").FindId(dataStorage.Starting_from_location_id).One(&response); err != nil {
			// w.WriteHeader(404)
			return
		}
		endLat = response.Coordinate.Lat
		endLng = response.Coordinate.Lng
		uberAPI(&carRes, dataStorage, startLat, startLng, endLat, endLng)
		carRes.Status = "requesting"
		carRes.Starting_from_location_id = dataStorage.Best_route_location_id[len(dataStorage.Best_route_location_id)-1]
		carRes.Next_destination_location_id = dataStorage.Starting_from_location_id
	} else if dataStorage.Index > len(dataStorage.Best_route_location_id) {
		carRes.Status = "finished"
		carRes.Starting_from_location_id = dataStorage.Starting_from_location_id
		carRes.Next_destination_location_id = dataStorage.Starting_from_location_id
	} else {
		if err := session.DB("cmpe273").C("assignment2").FindId(dataStorage.Best_route_location_id[dataStorage.Index-1]).One(&response); err != nil {
			// w.WriteHeader(404)
			return
		}
		startLat = response.Coordinate.Lat
		startLng = response.Coordinate.Lng

		if err := session.DB("cmpe273").C("assignment2").FindId(dataStorage.Best_route_location_id[dataStorage.Index]).One(&response); err != nil {
			// w.WriteHeader(404)
			return
		}
		endLat = response.Coordinate.Lat
		endLng = response.Coordinate.Lng
		uberAPI(&carRes, dataStorage, startLat, startLng, endLat, endLng)
		carRes.Status = "requesting"
		carRes.Starting_from_location_id = dataStorage.Best_route_location_id[dataStorage.Index-1]
		carRes.Next_destination_location_id = dataStorage.Best_route_location_id[dataStorage.Index]
	}

	carRes.Id = dataStorage.Id
	carRes.Best_route_location_id = dataStorage.Best_route_location_id
	carRes.Total_uber_costs = dataStorage.Total_uber_costs
	carRes.Total_uber_duration = dataStorage.Total_uber_duration
	carRes.Total_distance = dataStorage.Total_distance

	dataStorage.Index = dataStorage.Index + 1
	hmap[dataStorage.Id] = dataStorage

	trip, _ := json.Marshal(carRes)
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", trip)
}

func Delete(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}
	userId := bson.ObjectIdHex(id)

	if err := session.DB("cmpe273").C("assignment2").RemoveId(userId); err != nil {
		w.WriteHeader(404)
		return
	}
	w.WriteHeader(200)
}

func main() {
	id = 0
	hmap = make(map[int]DataStorage)

	session, _ = mgo.Dial("mongodb://admin:12345@ds051838.mongolab.com:51838/cmpe273")

	defer session.Close()

	mux := httprouter.New()
	//connectDB()
	mux.GET("/locations/:id", Get)
	mux.GET("/trips/:trip_id", GetTrip)
	mux.POST("/locations/", Create)
	mux.POST("/trips/", CreateTrip)
	mux.DELETE("/locations/:id", Delete)
	mux.PUT("/locations/:id", Update)
	mux.PUT("/trips/:trip_id/request", CarRequest)
	server := http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}
	server.ListenAndServe()
}

func createURL(address string, city string, state string) string {
	var getURL string
	spStr := strings.Split(address, " ")
	for i := 0; i < len(spStr); i++ {
		if i == 0 {
			getURL = spStr[i] + "+"
		} else if i == len(spStr)-1 {
			getURL = getURL + spStr[i] + ","
		} else {
			getURL = getURL + spStr[i] + "+"
		}
	}
	spStr = strings.Split(city, " ")
	for i := 0; i < len(spStr); i++ {
		if i == 0 {
			getURL = getURL + "+" + spStr[i]
		} else if i == len(spStr)-1 {
			getURL = getURL + "+" + spStr[i] + ","
		} else {
			getURL = getURL + "+" + spStr[i]
		}
	}
	getURL = getURL + "+" + state
	return getURL
}

func getLocation(response *Response, formatURL string) {
	urlLeft := "http://maps.google.com/maps/api/geocode/json?address="
	urlRight := "&sensor=false"
	urlFormat := urlLeft + formatURL + urlRight

	getLocation, err := http.Get(urlFormat)
	if err != nil {
		fmt.Println("Get Location Error", err)
		panic(err)
	}

	body, err := ioutil.ReadAll(getLocation.Body)
	if err != nil {
		fmt.Println("Get Location Error", err)
		panic(err)
	}

	var data MyJsonName
	byt := []byte(body)
	if err := json.Unmarshal(byt, &data); err != nil {
		panic(err)
	}
	response.Coordinate.Lat = data.Results[0].Geometry.Location.Lat
	response.Coordinate.Lng = data.Results[0].Geometry.Location.Lng
}

func getBestRoute(tripRes *TripResponse, dataStorage *DataStorage, originId bson.ObjectId, targetId []bson.ObjectId) {
	pmtTarget, err := permutation.NewPerm(targetId, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	res := make([][]bson.ObjectId, 0, 0)
	routePrice := make([]int, 0, 0)
	routeDuration := make([]int, 0, 0)
	routeDistance := make([]float64, 0, 0)
	curPrice := 0
	curDuration := 0
	curDistance := 0.0
	for result, err := pmtTarget.Next(); err == nil; result, err = pmtTarget.Next() {
		//----------------------get Price---------------------------------
		for i := 0; i <= len(result.([]bson.ObjectId)); i++ {
			var startLat float64
			var startLng float64
			var endLat float64
			var endLng float64
			minPrice := 0
			minDuration := 0
			minDistance := 0.0
			response := Response{}
			if i == 0 {
				if err := session.DB("cmpe273").C("assignment2").FindId(originId).One(&response); err != nil {
					// w.WriteHeader(404)
					return
				}
				startLat = response.Coordinate.Lat
				startLng = response.Coordinate.Lng

				if err := session.DB("cmpe273").C("assignment2").FindId((result.([]bson.ObjectId))[i]).One(&response); err != nil {
					// w.WriteHeader(404)
					return
				}
				endLat = response.Coordinate.Lat
				endLng = response.Coordinate.Lng
			} else if i == len(result.([]bson.ObjectId)) {
				if err := session.DB("cmpe273").C("assignment2").FindId((result.([]bson.ObjectId))[i-1]).One(&response); err != nil {
					// w.WriteHeader(404)
					return
				}
				startLat = response.Coordinate.Lat
				startLng = response.Coordinate.Lng

				if err := session.DB("cmpe273").C("assignment2").FindId(originId).One(&response); err != nil {
					// w.WriteHeader(404)
					return
				}
				endLat = response.Coordinate.Lat
				endLng = response.Coordinate.Lng
			} else {
				if err := session.DB("cmpe273").C("assignment2").FindId((result.([]bson.ObjectId))[i-1]).One(&response); err != nil {
					// w.WriteHeader(404)
					return
				}
				startLat = response.Coordinate.Lat
				startLng = response.Coordinate.Lng

				if err := session.DB("cmpe273").C("assignment2").FindId((result.([]bson.ObjectId))[i]).One(&response); err != nil {
					// w.WriteHeader(404)
					return
				}
				endLat = response.Coordinate.Lat
				endLng = response.Coordinate.Lng
			}

			// you need to add your sever token at the end
			urlLeft := "https://api.uber.com/v1/estimates/price?"
			urlRight := "start_latitude=" + strconv.FormatFloat(startLat, 'f', -1, 64) + "&start_longitude=" + strconv.FormatFloat(startLng, 'f', -1, 64) + "&end_latitude=" + strconv.FormatFloat(endLat, 'f', -1, 64) + "&end_longitude=" + strconv.FormatFloat(endLng, 'f', -1, 64) + "&server_token=xxxxxxxxxxxxxxxxxxxxxxx"
			urlFormat := urlLeft + urlRight

			getPrices, err := http.Get(urlFormat)
			if err != nil {
				fmt.Println("Get Prices Error", err)
				panic(err)
			}

			var data UberAPIResponse
			json.NewDecoder(getPrices.Body).Decode(&data)
			minPrice = data.Prices[0].LowEstimate
			minDuration = data.Prices[0].Duration
			minDistance = data.Prices[0].Distance
			for i := 0; i < len(data.Prices); i++ {
				if minPrice > data.Prices[i].LowEstimate && data.Prices[i].LowEstimate > 0 {
					minPrice = data.Prices[i].LowEstimate
					minDuration = data.Prices[i].Duration
					minDistance = data.Prices[i].Distance
				}
			}
			curPrice = curPrice + minPrice
			curDuration = curDuration + minDuration
			curDistance = curDistance + minDistance
		}
		//----------------------get Price---------------------------------
		routePrice = AppendInt(routePrice, curPrice)
		routeDuration = AppendInt(routeDuration, curDuration)
		routeDistance = AppendFloat(routeDistance, curDistance)
		fmt.Println(curPrice)
		fmt.Println(curDuration)
		fmt.Println(curDistance)
		curPrice = 0
		curDuration = 0
		curDistance = 0.0
		res = AppendBsonId(res, result.([]bson.ObjectId))
		fmt.Println(pmtTarget.Index(), result.([]bson.ObjectId))
	}
	index := 0
	curPrice = 1000
	for i := 0; i < len(routePrice); i++ {
		if curPrice > routePrice[i] {
			curPrice = routePrice[i]
			index = i
		}
	}
	fmt.Println("best route is => ")
	fmt.Println(res[index])
	fmt.Println(routePrice[index])
	fmt.Println(routeDuration[index])
	fmt.Println(routeDistance[index])
	// res[index] // 最后的数组
	tripRes.Best_route_location_id = res[index]
	tripRes.Total_distance = routeDistance[index]
	tripRes.Total_uber_costs = routePrice[index]
	tripRes.Total_uber_duration = routeDuration[index]
	dataStorage.Id = tripRes.Id
	dataStorage.Index = 0
	dataStorage.Status = tripRes.Status
	dataStorage.Starting_from_location_id = tripRes.Starting_from_location_id
	dataStorage.Best_route_location_id = res[index]
	dataStorage.Total_uber_costs = tripRes.Total_uber_costs
	dataStorage.Total_uber_duration = tripRes.Total_uber_duration
	dataStorage.Total_distance = tripRes.Total_distance
}

func AppendBsonId(slice [][]bson.ObjectId, data ...[]bson.ObjectId) [][]bson.ObjectId {
	m := len(slice)
	n := m + 1
	if n > cap(slice) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newSlice := make([][]bson.ObjectId, (n+1)*2)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0:n]
	copy(slice[m:n], data)
	return slice
}

func AppendInt(slice []int, data ...int) []int {
	m := len(slice)
	n := m + 1
	if n > cap(slice) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newSlice := make([]int, (n+1)*2)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0:n]
	copy(slice[m:n], data)
	return slice
}

func AppendFloat(slice []float64, data ...float64) []float64 {
	m := len(slice)
	n := m + 1
	if n > cap(slice) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newSlice := make([]float64, (n+1)*2)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0:n]
	copy(slice[m:n], data)
	return slice
}

func uberAPI(carRes *CarResponse, dataStorage DataStorage, startLat float64, startLng float64, endLat float64, endLng float64) {
	minPrice := 0
	serverToken := "xxxxxxxxxxxxxxxxxxxxxxx"
	urlLeft := "https://api.uber.com/v1/estimates/price?"
	urlRight := "start_latitude=" + strconv.FormatFloat(startLat, 'f', -1, 64) + "&start_longitude=" + strconv.FormatFloat(startLng, 'f', -1, 64) + "&end_latitude=" + strconv.FormatFloat(endLat, 'f', -1, 64) + "&end_longitude=" + strconv.FormatFloat(endLng, 'f', -1, 64) + "&server_token=" + serverToken
	urlFormat := urlLeft + urlRight
	var userrequest UserRequest

	getPrices, err := http.Get(urlFormat)
	if err != nil {
		fmt.Println("Get Prices Error", err)
		panic(err)
	}

	var data UberAPIResponse
	index := 0

	json.NewDecoder(getPrices.Body).Decode(&data)

	minPrice = data.Prices[0].LowEstimate
	for i := 0; i < len(data.Prices); i++ {
		if minPrice > data.Prices[i].LowEstimate {
			minPrice = data.Prices[i].LowEstimate
			index = i
		}
		userrequest.Product_id = data.Prices[index].ProductID
	}

	urlPath := "https://sandbox-api.uber.com/v1/requests"
	userrequest.Start_latitude = startLat
	userrequest.Start_longitude = startLng
	userrequest.End_latitude = endLat
	userrequest.End_longitude = endLng
	accessToken := "xxxxxxxxxxxxxxxxxxxx"

	requestbody, _ := json.Marshal(userrequest)
	client := &http.Client{}
	req, err := http.NewRequest("POST", urlPath, bytes.NewBuffer(requestbody))
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+accessToken)
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("QueryInfo: http.Get", err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	uberRes := UberResponse{}
	json.Unmarshal(body, &uberRes)

	fmt.Println(uberRes)

	carRes.Uber_wait_time_eta = uberRes.Eta
}

func getID() int {
	if id == 0 {
		for id == 0 {
			id = rand.Intn(10000)
		}
	} else {
		id = id + 1
	}
	return id
}
