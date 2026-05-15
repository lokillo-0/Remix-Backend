package fortnite

import (
	_ "embed"
	"encoding/json"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

//go:embed locker_base.json
var lockerBaseRaw []byte

func getLocker(accountId string, deploymentId string) (*accounts.Locker, error) {
	key := accountId + ":locker:v4"

	var locker accounts.Locker
	err := odin.Find("Accounts_LockerV4", key, &locker)

	if err != nil {
		if err := json.Unmarshal(lockerBaseRaw, &locker); err != nil {
			return nil, err
		}

		now := time.Now().UTC().Format(time.RFC3339)

		locker.ActiveLoadoutGroup.AccountId = accountId
		locker.ActiveLoadoutGroup.DeploymentId = deploymentId
		locker.ActiveLoadoutGroup.CreationTime = now
		locker.ActiveLoadoutGroup.UpdatedTime = now
		locker.Bucket.ID = key
		locker.ID = key

		locker.Bucket.Save(locker)
		return &locker, nil
	}

	return &locker, nil
}

func GETLockerItems(c *gin.Context) {
	accountId := c.Param("accountId")
	deploymentId := c.Param("deploymentId")

	locker, err := getLocker(accountId, deploymentId)
	if err != nil {
		utilities.Basic.NotAcceptable().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	locker.ActiveLoadoutGroup.AccountId = accountId
	locker.ActiveLoadoutGroup.DeploymentId = deploymentId
	locker.ActiveLoadoutGroup.UpdatedTime = now

	locker.Bucket.Save(locker)

	c.JSON(200, locker)
}

func PUTActiveLoadoutGroup(c *gin.Context) {
	accountId := c.Param("accountId")
	deploymentId := c.Param("deploymentId")

	var body struct {
		EquippedPresetId *string                           `json:"equippedPresetId"`
		Loadouts         map[string]accounts.LoadoutSchema `json:"loadouts"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.Basic.NotAcceptable().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	locker, err := getLocker(accountId, deploymentId)
	if err != nil {
		utilities.Basic.NotAcceptable().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	if body.EquippedPresetId != nil {
		raw, _ := json.Marshal(locker.ActiveLoadoutGroup)
		var tmp map[string]interface{}
		json.Unmarshal(raw, &tmp)

		tmp["equippedPresetId"] = *body.EquippedPresetId

		newRaw, _ := json.Marshal(tmp)
		json.Unmarshal(newRaw, &locker.ActiveLoadoutGroup)
	}

	locker.ActiveLoadoutGroup.Loadouts = body.Loadouts
	locker.ActiveLoadoutGroup.UpdatedTime = now

	locker.Bucket.Save(locker)

	c.JSON(200, locker.ActiveLoadoutGroup)
}
