package server

import (
	"net/http"
	"strings"

	citylogic "wplink/backend/app/internal/logic/city"
	"wplink/backend/common/response"
)

type CityAPIStore interface {
	citylogic.CityStationStore
	citylogic.ResourceTypeStore
}

func NewAPIRouter(store CityAPIStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/city-stations", func(w http.ResponseWriter, r *http.Request) {
		resp, err := citylogic.NewListCityStationsLogic(store).ListCityStations(r.Context())
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/city-stations/", func(w http.ResponseWriter, r *http.Request) {
		cityCode, ok := cityCodeFromResourceTypePath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		resp, err := citylogic.NewListResourceTypesLogic(store).ListResourceTypes(r.Context(), cityCode)
		response.JSON(w, resp, err)
	})
	return mux
}

func cityCodeFromResourceTypePath(requestPath string) (string, bool) {
	const prefix = "/api/v1/city-stations/"
	const suffix = "/resource-types"
	if !strings.HasPrefix(requestPath, prefix) || !strings.HasSuffix(requestPath, suffix) {
		return "", false
	}
	cityCode := strings.TrimSuffix(strings.TrimPrefix(requestPath, prefix), suffix)
	cityCode = strings.Trim(cityCode, "/")
	return cityCode, cityCode != ""
}
