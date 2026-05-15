package models

type Requirement struct {
	RequirementType string `json:"requirementType"`
	RequiredId      string `json:"requiredId"`
	MinQuantity     int    `json:"minQuantity"`
}
