package accounts

import "github.com/andr1ww/odin"

type Locker struct {
	odin.Bucket `bucket:"Accounts_LockerV4" database:"xenon_profiles"`

	ActiveLoadoutGroup  ActiveLoadoutGroup   `json:"activeLoadoutGroup"`
	LoadoutGroupPresets []LoadoutGroupPreset `json:"loadoutGroupPresets"`
	LoadoutPresets      []LoadoutPreset      `json:"loadoutPresets"`
}

type LoadoutPreset struct {
	DeploymentId         string        `json:"deploymentId"`
	AccountId            string        `json:"accountId"`
	LoadoutType          string        `json:"loadoutType"`
	PresetId             string        `json:"presetId"`
	PresetIndex          int           `json:"presetIndex"`
	AthenaItemId         string        `json:"athenaItemId"`
	CreationTime         string        `json:"creationTime"`
	UpdatedTime          string        `json:"updatedTime"`
	LoadoutSlots         []LoadoutSlot `json:"loadoutSlots"`
	DisplayName          string        `json:"displayName"`
	PresetFavoriteStatus string        `json:"presetFavoriteStatus"`
}

type LoadoutGroupPreset struct {
	AccountId            string                   `json:"accountId"`
	AthenaItemId         string                   `json:"athenaItemId"`
	CreationTime         string                   `json:"creationTime"`
	DeploymentId         string                   `json:"deploymentId"`
	DisplayName          string                   `json:"displayName"`
	Loadouts             map[string]LoadoutSchema `json:"loadouts"`
	PresetFavoriteStatus string                   `json:"presetFavoriteStatus"`
	PresetId             string                   `json:"presetId"`
	PresetIndex          int                      `json:"presetIndex"`
}

type LoadoutSchema struct {
	LoadoutSlots []LoadoutSlot `json:"loadoutSlots"`
}

type LoadoutSlot struct {
	SlotTemplate       string              `json:"slotTemplate"`
	EquippedItemId     string              `json:"equippedItemId,omitempty"`
	ItemCustomizations []ItemCustomization `json:"itemCustomizations"`
}

type ItemCustomization struct {
	ChannelTag     string `json:"channelTag"`
	VariantTag     string `json:"variantTag"`
	AdditionalData string `json:"additionalData"`
}

type ActiveLoadoutGroup struct {
	AccountId    string                   `json:"accountId"`
	DeploymentId string                   `json:"deploymentId"`
	AthenaItemId string                   `json:"athenaItemId"`
	CreationTime string                   `json:"creationTime"`
	UpdatedTime  string                   `json:"updatedTime"`
	Loadouts     map[string]LoadoutSchema `json:"loadouts"`
	ShuffleType  string                   `json:"shuffleType"`
}
