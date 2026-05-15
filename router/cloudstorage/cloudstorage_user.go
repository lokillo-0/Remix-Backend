package cloudstorage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/utilities"
)

func Init() {
	cfg := map[string]string{
		"AccessKeyId": utilities.Get[string]("cf_access_id"), "SecretAccessKey": utilities.Get[string]("cf_secret_key"),
		"Region": utilities.Get[string]("cf_region"), "Endpoint": utilities.Get[string]("cf_endpoint"),
	}
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg["AccessKeyId"], cfg["SecretAccessKey"], "")),
		config.WithRegion(cfg["Region"]),
		config.WithEndpointResolver(aws.EndpointResolverFunc(func(string, string) (aws.Endpoint, error) {
			return aws.Endpoint{URL: cfg["Endpoint"], HostnameImmutable: true}, nil
		})),
	)
	if err != nil {
		panic(fmt.Sprintf("Unable to load AWS SDK config: %v", err))
	}
	utilities.CC = s3.NewFromConfig(awsCfg)
}

func validateAndParse(c *gin.Context) (string, string, bool) {
	ua := utilities.Parse(c.GetHeader("User-Agent"))
	if ua == nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return "", "", false
	}
	return c.Param("accountId"), strconv.Itoa(ua.Season), true
}

func getS3Object(accountId, seasonStr string) (*s3.GetObjectOutput, error) {
	key := fmt.Sprintf("Settings/%s/ClientSettings-%s.Sav", accountId, seasonStr)
	result, err := utilities.CC.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String("remix"),
		Key:    aws.String(key),
	})

	return result, err
}

func GetUsersCloudstorageFiles2(c *gin.Context) {
	if c.Param("accountId") == "config" {
		c.Status(http.StatusNoContent)
		return
	}

	c.Header("Content-Type", "application/json")
	accountId, seasonStr, ok := validateAndParse(c)
	if !ok {
		return
	}

	isAndroid := strings.Contains(strings.ToLower(c.Request.UserAgent()), "android")
	isIOS := strings.Contains(strings.ToLower(c.Request.UserAgent()), "ios")

	var result *s3.GetObjectOutput
	var err error

	if isAndroid {
		androidKey := fmt.Sprintf("Settings/%s/ClientSettings-%s-Android.Sav", accountId, seasonStr)
		result, err = utilities.CC.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String("remix"),
			Key:    aws.String(androidKey),
		})
		if err != nil {
			c.JSON(http.StatusOK, []gin.H{})
			return
		}
	} else if isIOS {
		iosKey := fmt.Sprintf("Settings/%s/ClientSettings-%s-IOS.Sav", accountId, seasonStr)
		result, err = utilities.CC.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String("remix"),
			Key:    aws.String(iosKey),
		})
		if err != nil {
			c.JSON(http.StatusOK, []gin.H{})
			return
		}
	} else {
		result, err = getS3Object(accountId, seasonStr)
		if err != nil {
			c.JSON(http.StatusOK, []gin.H{})
			return
		}
	}

	defer result.Body.Close()

	fileBytes, err := io.ReadAll(result.Body)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	sha1Hash, sha256Hash := sha1.Sum(fileBytes), sha256.Sum256(fileBytes)

	resp := gin.H{
		"uniqueFilename": "ClientSettings.Sav",
		"filename":       "ClientSettings.Sav",
		"hash":           hex.EncodeToString(sha1Hash[:]),
		"hash256":        hex.EncodeToString(sha256Hash[:]),
		"length":         len(fileBytes),
		"contentType":    "application/octet-stream",
		"uploaded":       result.LastModified,
		"storageType":    "S3",
		"accountId":      accountId,
	}

	if isAndroid {
		resp["uniqueFilename"] = "ClientSettingsAndroid.Sav"
		resp["filename"] = "ClientSettingsAndroid.Sav"
		resp["contentType"] = "application/octet-stream"
		resp["storageIds"] = map[string]interface{}{}
		resp["metadata"] = map[string]interface{}{}
	} else if isIOS {
		resp["uniqueFilename"] = "ClientSettingsIOS.Sav"
		resp["filename"] = "ClientSettingsIOS.Sav"
		resp["contentType"] = "application/octet-stream"
		resp["storageIds"] = map[string]interface{}{}
		resp["metadata"] = map[string]interface{}{}
	}

	c.JSON(http.StatusOK, []gin.H{resp})
}

func GetUserCloudstorageFileViaAccess(c *gin.Context) {
	accountId := c.Param("accountId")
	rawFilename := strings.TrimPrefix(c.Param("filename"), "/")

	dashIdx := strings.Index(rawFilename, "-")
	if dashIdx == -1 {
		c.Status(http.StatusNotFound)
		return
	}
	filename := rawFilename[dashIdx+1:]

	ua := utilities.Parse(c.GetHeader("User-Agent"))
	if ua == nil {
		c.Status(http.StatusNotFound)
		return
	}
	seasonStr := strconv.Itoa(ua.Season)

	var key string
	if strings.Contains(strings.ToLower(filename), "android") {
		key = fmt.Sprintf("Settings/%s/ClientSettings-%s-Android.Sav", accountId, seasonStr)
	} else {
		key = fmt.Sprintf("Settings/%s/ClientSettings-%s.Sav", accountId, seasonStr)
	}

	result, err := utilities.CC.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String("remix"),
		Key:    aws.String(key),
	})
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer result.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, result.Body); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", buf.Bytes())
}

func GetUserCloudstorageFile(c *gin.Context) {
	filename := c.Param("filename")
	if filename != "ClientSettings.Sav" && filename != "ClientSettingsIOS.Sav" && filename != "ClientSettingsAndroid.Sav" {
		c.Status(http.StatusOK)
		return
	}

	accountId, seasonStr, ok := validateAndParse(c)
	if !ok {
		return
	}

	var key string
	if strings.Contains(strings.ToLower(filename), "android") {
		key = fmt.Sprintf("Settings/%s/ClientSettings-%s-Android.Sav", accountId, seasonStr)
	} else if strings.Contains(strings.ToLower(filename), "ios") {
		key = fmt.Sprintf("Settings/%s/ClientSettings-%s-IOS.Sav", accountId, seasonStr)
	} else {
		key = fmt.Sprintf("Settings/%s/ClientSettings-%s.Sav", accountId, seasonStr)
	}

	result, err := utilities.CC.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String("remix"),
		Key:    aws.String(key),
	})
	if err != nil {
		utilities.MCP.InvalidPayload().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}
	defer result.Body.Close()

	contentType := "application/octet-stream"
	c.Header("Content-Type", contentType)

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, result.Body); err != nil {
		utilities.MCP.InvalidPayload().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}
	c.Data(http.StatusOK, contentType, buf.Bytes())
}

func SaveUsersCloudstorageFile(c *gin.Context) {
	filename := c.Param("filename")
	if filename != "ClientSettings.Sav" && filename != "ClientSettingsIOS.Sav" && filename != "ClientSettingsAndroid.Sav" {
		c.Status(http.StatusBadRequest)
		return
	}

	accountId, seasonStr, ok := validateAndParse(c)
	if !ok {
		return
	}

	fileBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	var key string
	baseFilename := strings.TrimSuffix(filename, ".Sav")
	contentType := "application/octet-stream"

	if strings.Contains(strings.ToLower(filename), "android") {
		key = fmt.Sprintf("Settings/%s/ClientSettings-%s-Android.Sav", accountId, seasonStr)
		contentType = "text/plain"
	} else if strings.Contains(strings.ToLower(filename), "ios") {
		key = fmt.Sprintf("Settings/%s/ClientSettings-%s-IOS.Sav", accountId, seasonStr)
	} else {
		key = fmt.Sprintf("Settings/%s/%s-%s.Sav", accountId, baseFilename, seasonStr)
	}

	_, err = utilities.CC.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String("remix"),
		Key:           aws.String(key),
		Body:          bytes.NewReader(fileBytes),
		ContentLength: aws.Int64(int64(len(fileBytes))),
		ContentType:   aws.String(contentType),
	})

	if err != nil {
		c.Status(http.StatusInternalServerError)
	} else {
		c.Status(http.StatusOK)
	}
}
