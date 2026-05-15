package models

type Item struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Set  struct {
		Value        string `json:"value"`
		Text         string `json:"text"`
		BackendValue string `json:"backendValue"`
	} `json:"set"`
	Item struct {
		Value        string `json:"value"`
		Text         string `json:"text"`
		BackendValue string `json:"backendValue"`
	} `json:"item"`
	Images struct {
		Icon      string `json:"icon"`
		SmallIcon string `json:"smallIcon"`
	} `json:"images"`
	Introduction struct {
		BackendValue int `json:"backendValue"`
	} `json:"introduction"`
}
