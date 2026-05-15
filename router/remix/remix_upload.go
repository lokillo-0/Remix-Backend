package remix

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/andr1ww/odin"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func POSTRemixUploadProfile(c *gin.Context) {
	const maxSize = 2 * 1024 * 1024

	accountid := c.Param("accountid")

	tokenHeader := c.GetHeader("Authorization")
	if tokenHeader == "" {
		utilities.Authentication.InvalidHeader().Apply(c.Writer)
		return
	}
	token := tokenHeader
	if len(tokenHeader) > 7 && (tokenHeader[:7] == "bearer " || tokenHeader[:7] == "Bearer ") {
		token = tokenHeader[7:]
	}
	sess, _ := odin.FindWhere("Accounts_Sessions", map[string]interface{}{"token": token}, func() interface{} {
		return &accounts.Session{}
	})
	if len(sess) == 0 || sess[0].(*accounts.Session).AccountID != accountid {
		utilities.Authentication.InvalidToken().Apply(c.Writer)
		return
	}

	fileBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, maxSize+1))
	if err != nil {
		utilities.Internal.ServerError().WithMessage("failed to read profile picture").Apply(c.Writer)
		return
	}

	if len(fileBytes) > maxSize {
		utilities.Internal.ServerError().WithMessage("file too large, must be <= 2MB").Apply(c.Writer)
		return
	}

	contentType := http.DetectContentType(fileBytes)
	if contentType != "image/png" {
		utilities.Internal.ServerError().WithMessage("invalid content type, expected image/png").Apply(c.Writer)
		return
	}

	file := bytes.NewReader(fileBytes)

	var account accounts.Account
	if err := odin.Find("Accounts", accountid, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		return
	}

	_, err = utilities.CC.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String("remix"),
		Key:         aws.String(fmt.Sprintf("Data/%s/%s.png", accountid, accountid)),
		Body:        file,
		ContentType: aws.String("image/png"),
	})
	if err != nil {
		utilities.Internal.ServerError().WithMessage(fmt.Sprintf("failed to upload profile picture: %v", err)).Apply(c.Writer)
		return
	}

	account.ProfilePicture = fmt.Sprintf("https://saturn.nxa.app/Data/%s/%s.png", accountid, accountid)
	account.Bucket.Save(account)

	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(http.StatusOK)
	fmt.Fprintf(c.Writer, "https://saturn.nxa.app/Data/%s/%s.png", accountid, accountid)
}
