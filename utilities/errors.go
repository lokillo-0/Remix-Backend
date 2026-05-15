package utilities

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

type ResponseBody struct {
	ErrorCode          string                 `json:"errorCode"`
	ErrorMessage       string                 `json:"errorMessage"`
	MessageVars        []string               `json:"messageVars,omitempty"`
	NumericErrorCode   int                    `json:"numericErrorCode"`
	OriginatingService string                 `json:"originatingService"`
	Intent             string                 `json:"intent"`
	ValidationFailures map[string]interface{} `json:"validationFailures,omitempty"`
}

type Intent string

const (
	ProdLive Intent = "ProdLive"
	Prod     Intent = "Prod"
	Live     Intent = "Live"
)

type ApiError struct {
	StatusCode int
	Response   *ResponseBody
}

func NewApiError(code, message, service string, numeric, statusCode int, messageVariables ...string) *ApiError {
	var vars []string
	if len(messageVariables) > 0 {
		vars = messageVariables
	}

	return &ApiError{
		StatusCode: statusCode,
		Response: &ResponseBody{
			ErrorCode:          code,
			ErrorMessage:       message,
			MessageVars:        vars,
			NumericErrorCode:   numeric,
			OriginatingService: service,
			Intent:             "Xenon",
		},
	}
}

func (e *ApiError) WithMessageVar(variables []string) *ApiError {
	e.Response.MessageVars = variables
	return e
}

func (e *ApiError) WithIntent(intent Intent) *ApiError {
	e.Response.Intent = string(intent)
	return e
}

func (e *ApiError) WithMessage(message string) *ApiError {
	e.Response.ErrorMessage = message
	return e
}

func (e *ApiError) Variable(variables []string) *ApiError {
	re := regexp.MustCompile(`\{(\d+)\}`)
	matches := re.FindAllStringSubmatch(e.Response.ErrorMessage, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		var placeholderIndex int
		fmt.Sscanf(match[1], "%d", &placeholderIndex)
		if placeholderIndex < len(variables) {
			e.Response.ErrorMessage = regexp.MustCompile(fmt.Sprintf(`\{%d\}`, placeholderIndex)).
				ReplaceAllString(e.Response.ErrorMessage, variables[placeholderIndex])
		}
	}
	return e
}

func (e *ApiError) OriginatingService(service string) *ApiError {
	e.Response.OriginatingService = service
	return e
}

func (e *ApiError) With(messageVariables ...string) *ApiError {
	if e.Response.MessageVars == nil {
		e.Response.MessageVars = make([]string, 0)
	}
	e.Response.MessageVars = append(e.Response.MessageVars, messageVariables...)
	return e
}

func (e *ApiError) Apply(w http.ResponseWriter) {
	if w == nil || e == nil {
		return
	}

	e.Response.ErrorMessage = e.GetMessage()

	defer func() {
		_ = recover()
	}()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Epic-Error-Code", fmt.Sprintf("%d", e.Response.NumericErrorCode))
	w.Header().Set("X-Epic-Error-Name", e.Response.ErrorCode)
	w.WriteHeader(e.StatusCode)
	_ = json.NewEncoder(w).Encode(e.Response)
}

func (e *ApiError) ApplyC(w http.ResponseWriter, c *gin.Context) {
	e.Response.ErrorMessage = e.GetMessage()
	if c != nil {
		c.AbortWithStatusJSON(e.StatusCode, e.Response)
		return
	}
	if w != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Epic-Error-Code", fmt.Sprintf("%d", e.Response.NumericErrorCode))
		w.Header().Set("X-Epic-Error-Name", e.Response.ErrorCode)
		w.WriteHeader(e.StatusCode)
		json.NewEncoder(w).Encode(e.Response)
		return
	}
}

func (e *ApiError) GetMessage() string {
	if len(e.Response.MessageVars) == 0 {
		return e.Response.ErrorMessage
	}

	result := e.Response.ErrorMessage
	for i, v := range e.Response.MessageVars {
		result = regexp.MustCompile(fmt.Sprintf(`\{%d\}`, i)).ReplaceAllString(result, v)
	}
	return result
}

func (e *ApiError) ShortenedError() string {
	return fmt.Sprintf("%s - %s", e.Response.ErrorCode, e.Response.ErrorMessage)
}

func (e *ApiError) ThrowHttpException() error {
	return &ApiException{
		StatusCode: e.StatusCode,
		Response:   *e.Response,
	}
}

func (e *ApiError) DevMessage(message, devMode string) *ApiError {
	if devMode != "true" {
		return e
	}
	e.Response.ErrorMessage += fmt.Sprintf(" (Dev: -%s-)", message)
	return e
}

type ApiException struct {
	StatusCode int
	Response   ResponseBody
}

func (e *ApiException) Error() string {
	return fmt.Sprintf("%s: %s", e.Response.ErrorCode, e.Response.ErrorMessage)
}

var (
	Storefront = struct {
		InvalidItem          func() *ApiError
		CurrencyInsufficient func() *ApiError
		HasAllItems          func() *ApiError
		AlreadyOwned         func() *ApiError
	}{
		InvalidItem: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.fortnite.invalid_item_id",
				"Failed to get item from the current shop.",
				"com.remix.storefront",
				1040,
				http.StatusBadRequest,
			)
		},
		CurrencyInsufficient: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.currency.mtx.insufficient",
				"You can not afford this item.",
				"com.remix.storefront",
				1040,
				http.StatusBadRequest,
			)
		},
		HasAllItems: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.offer.has_all_items",
				"You already own every item.",
				"com.remix.storefront",
				1040,
				http.StatusBadRequest,
			)
		},
		AlreadyOwned: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.offer.already_owned",
				"You already own this item.",
				"com.remix.storefront",
				1040,
				http.StatusBadRequest,
			)
		},
	}

	Authentication = struct {
		InvalidHeader        func() *ApiError
		MissingPermission    func() *ApiError
		InvalidRequest       func() *ApiError
		InvalidToken         func() *ApiError
		WrongGrantType       func() *ApiError
		NotYourAccount       func() *ApiError
		ValidationFailed     func() *ApiError
		AuthenticationFailed func() *ApiError
		NotOwnSessionRemoval func() *ApiError
		UnknownSession       func() *ApiError
		UsedClientToken      func() *ApiError
		OAuth                struct {
			InvalidBody                func() *ApiError
			UnsupportedGrant           func() *ApiError
			InvalidExternalAuthType    func() *ApiError
			GrantNotImplemented        func() *ApiError
			TooManySessions            func() *ApiError
			InvalidAccountCredentials  func() *ApiError
			InvalidRefresh             func() *ApiError
			InvalidClient              func() *ApiError
			InvalidExchange            func() *ApiError
			ExpiredExchangeCodeSession func() *ApiError
			CorrectiveActionRequired   func() *ApiError
		}
	}{
		InvalidHeader: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.authentication.invalidHeader",
				"It looks like your authorization header is invalid or missing, please verify that you are sending the correct headers.",
				"com.remix.auth",
				1011,
				http.StatusBadRequest,
			)
		},
		MissingPermission: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.auth.missingPermission",
				"Sorry your login does not posses the permissions '{0}' needed to perform the requested operation",
				"com.remix.auth",
				12806,
				http.StatusForbidden,
			)
		},
		InvalidRequest: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.authentication.invalidRequest",
				"The request body you provided is either invalid or missing elements.",
				"com.remix.auth",
				1013,
				http.StatusBadRequest,
			)
		},
		InvalidToken: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.authentication.invalidToken",
				"Invalid token {0}",
				"com.remix.auth",
				1014,
				http.StatusUnauthorized,
			)
		},
		WrongGrantType: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.authentication.wrongGrantType",
				"Sorry, your client does not have the proper grant_type for access.",
				"com.remix.auth",
				1016,
				http.StatusBadRequest,
			)
		},
		NotYourAccount: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.authentication.notYourAccount",
				"You are not allowed to make changes to other people's accounts",
				"com.remix.auth",
				1023,
				http.StatusForbidden,
			)
		},
		ValidationFailed: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.authentication.validationFailed",
				"Sorry we couldn't validate your token {0}. Please try with a new token.",
				"com.remix.auth",
				1031,
				http.StatusUnauthorized,
			)
		},
		AuthenticationFailed: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.authentication.authenticationFailed",
				"Authentication failed for {0}",
				"com.remix.auth",
				1032,
				http.StatusUnauthorized,
			)
		},
		NotOwnSessionRemoval: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.authentication.notOwnSessionRemoval",
				"Sorry you cannot remove the auth session {0}. It was not issued to you.",
				"com.remix.auth",
				18040,
				http.StatusForbidden,
			)
		},
		UnknownSession: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.authentication.unknownSession",
				"Sorry we could not find the auth session {0}",
				"com.remix.auth",
				18051,
				http.StatusNotFound,
			)
		},
		UsedClientToken: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.authentication.wrongTokenType",
				"This route requires authentication via user access tokens, but you are using a client token",
				"com.remix.auth",
				18052,
				http.StatusUnauthorized,
			)
		},
		OAuth: struct {
			InvalidBody                func() *ApiError
			UnsupportedGrant           func() *ApiError
			InvalidExternalAuthType    func() *ApiError
			GrantNotImplemented        func() *ApiError
			TooManySessions            func() *ApiError
			InvalidAccountCredentials  func() *ApiError
			InvalidRefresh             func() *ApiError
			InvalidClient              func() *ApiError
			InvalidExchange            func() *ApiError
			ExpiredExchangeCodeSession func() *ApiError
			CorrectiveActionRequired   func() *ApiError
		}{
			InvalidBody: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.authentication.oauth.invalidBody",
					"The request body you provided is either invalid or missing elements.",
					"com.remix.auth",
					1013,
					http.StatusBadRequest,
				)
			},
			UnsupportedGrant: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.common.oauth.unsupported_grant_type",
					"Unsupported grant type: {0}",
					"com.remix.auth",
					1016,
					http.StatusBadRequest,
				)
			},
			InvalidExternalAuthType: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.authentication.oauth.invalidExternalAuthType",
					"The external auth type {0} you used is not supported by the server.",
					"com.remix.auth",
					1016,
					http.StatusBadRequest,
				)
			},
			GrantNotImplemented: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.authentication.grantNotImplemented",
					"The grant_type {0} you used is not supported by the server.",
					"com.remix.auth",
					1016,
					http.StatusNotImplemented,
				)
			},
			TooManySessions: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.authentication.oauth.tooManySessions",
					"Sorry too many sessions have been issued for your account. Please try again later",
					"com.remix.auth",
					18048,
					http.StatusBadRequest,
				)
			},
			InvalidAccountCredentials: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.authentication.oauth.invalidAccountCredentials",
					"Sorry the account credentials you are using are invalid",
					"com.remix.auth",
					18031,
					http.StatusBadRequest,
				)
			},
			InvalidRefresh: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.authentication.oauth.invalidRefresh",
					"The refresh token you provided is invalid.",
					"com.remix.auth",
					18036,
					http.StatusBadRequest,
				)
			},
			InvalidClient: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.authentication.oauth.invalidClient",
					"The client credentials you are using are invalid.",
					"com.remix.auth",
					18033,
					http.StatusForbidden,
				)
			},
			InvalidExchange: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.authentication.oauth.invalidExchange",
					"The exchange code {0} is invalid.",
					"com.remix.auth",
					18057,
					http.StatusBadRequest,
				)
			},
			ExpiredExchangeCodeSession: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.authentication.oauth.expiredExchangeCodeSession",
					"Sorry the originating session for the exchange code has expired.",
					"com.remix.auth",
					18128,
					http.StatusBadRequest,
				)
			},
			CorrectiveActionRequired: func() *ApiError {
				return NewApiError(
					"errors.com.epicgames.authentication.oauth.corrective_action_required",
					"Corrective action is required to continue.",
					"com.remix.auth",
					18206,
					http.StatusBadRequest,
				)
			},
		},
	}

	CloudStorage = struct {
		FileNotFound func() *ApiError
		FileTooLarge func() *ApiError
		InvalidAuth  func() *ApiError
	}{
		FileNotFound: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.cloudstorage.fileNotFound",
				"Cannot find the file you requested.",
				"com.remix.cloudstorage",
				12004,
				http.StatusNotFound,
			)
		},
		FileTooLarge: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.cloudstorage.fileTooLarge",
				"The file you are trying to upload is too large.",
				"com.remix.cloudstorage",
				12004,
				http.StatusRequestEntityTooLarge,
			)
		},
		InvalidAuth: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.cloudstorage.invalidAuth",
				"Invalid auth token.",
				"com.remix.cloudstorage",
				12004,
				http.StatusUnauthorized,
			)
		},
	}

	Account = struct {
		DisabledAccount       func() *ApiError
		InactiveAccount       func() *ApiError
		InvalidAccountIdCount func() *ApiError
		AccountNotFound       func() *ApiError
	}{
		DisabledAccount: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.account.disabledAccount",
				"Sorry, your account is disabled.",
				"com.remix.account",
				18001,
				http.StatusForbidden,
			)
		},
		InactiveAccount: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.account.account_not_active",
				"You have been permanently banned from Fortnite.",
				"com.remix.account",
				-1,
				http.StatusBadRequest,
			)
		},
		InvalidAccountIdCount: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.account.invalidAccountIdCount",
				"Sorry, the number of account IDs should be at least one and not more than 100.",
				"com.remix.account",
				18066,
				http.StatusBadRequest,
			)
		},
		AccountNotFound: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.account.accountNotFound",
				"Sorry, we couldn't find an account for {0}.",
				"com.remix.account",
				18007,
				http.StatusBadRequest,
			)
		},
	}

	MCP = struct {
		ProfileNotFound        func() *ApiError
		EmptyItems             func() *ApiError
		NotEnoughMtx           func() *ApiError
		WrongCommand           func() *ApiError
		OperationForbidden     func() *ApiError
		TemplateNotFound       func() *ApiError
		InvalidHeader          func() *ApiError
		InvalidPayload         func() *ApiError
		ItemNotFound           func() *ApiError
		WrongItemType          func(itemId, itemType string) *ApiError
		InvalidChatRequest     func() *ApiError
		OperationNotFound      func() *ApiError
		InvalidLockerSlotIndex func() *ApiError
		OutOfBounds            func() *ApiError
		InternalServerError    func() *ApiError
	}{
		ProfileNotFound: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.profileNotFound",
				"Sorry, we couldn't find a profile for {0}.",
				"com.remix.mcp",
				18007,
				http.StatusNotFound,
			)
		},
		EmptyItems: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.emptyItems",
				"No items found.",
				"com.remix.mcp",
				12700,
				http.StatusNotFound,
			)
		},
		NotEnoughMtx: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.notEnoughMtx",
				"Purchase: {0}: Required {1} MTX but account balance is only {2}.",
				"com.remix.mcp",
				12720,
				http.StatusBadRequest,
			)
		},
		WrongCommand: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.wrongCommand",
				"Wrong command.",
				"com.remix.mcp",
				12801,
				http.StatusBadRequest,
			)
		},
		OperationForbidden: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.operationForbidden",
				"Operation Forbidden.",
				"com.remix.mcp",
				12813,
				http.StatusForbidden,
			)
		},
		TemplateNotFound: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.templateNotFound",
				"Unable to find template configuration for profile.",
				"com.remix.mcp",
				12813,
				http.StatusNotFound,
			)
		},
		InvalidHeader: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.invalidHeader",
				"Parsing client revisions header failed.",
				"com.remix.mcp",
				12831,
				http.StatusBadRequest,
			)
		},
		InvalidPayload: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.invalidPayload",
				"Unable to parse command.",
				"com.remix.mcp",
				12806,
				http.StatusBadRequest,
			)
		},
		ItemNotFound: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.itemNotFound",
				"Locker item not found.",
				"com.remix.mcp",
				16006,
				http.StatusNotFound,
			)
		},
		WrongItemType: func(itemId, itemType string) *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.wrongItemType",
				fmt.Sprintf("Item %s is not a %s.", itemId, itemType),
				"com.remix.mcp",
				16009,
				http.StatusBadRequest,
			)
		},
		InvalidChatRequest: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.invalidChatRequest",
				"Invalid chat request.",
				"com.remix.mcp",
				16090,
				http.StatusBadRequest,
			)
		},
		OperationNotFound: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.operationNotFound",
				"Operation not found.",
				"com.remix.mcp",
				16035,
				http.StatusNotFound,
			)
		},
		InvalidLockerSlotIndex: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.InvalidLockerSlotIndex",
				"Invalid loadout index {0}, slot is empty.",
				"com.remix.mcp",
				16173,
				http.StatusBadRequest,
			)
		},
		OutOfBounds: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.mcp.outOfBounds",
				"Invalid loadout index (source: {0}, target: {1}).",
				"com.remix.mcp",
				16026,
				http.StatusBadRequest,
			)
		},
	}

	Matchmaking = struct {
		UnknownSession      func() *ApiError
		MissingCookie       func() *ApiError
		InvalidBucketId     func() *ApiError
		InvalidPartyPlayers func() *ApiError
		InvalidPlatform     func() *ApiError
		NotAllowedIngame    func() *ApiError
	}{
		UnknownSession: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.matchmaking.unknownSession",
				"Unknown session ID.",
				"com.remix.matchmaking",
				12101,
				http.StatusNotFound,
			)
		},
		MissingCookie: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.matchmaking.missingCookie",
				"Missing custom NetCL cookie.",
				"com.remix.matchmaking",
				1001,
				http.StatusBadRequest,
			)
		},
		InvalidBucketId: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.matchmaking.invalidBucketId",
				"Blank or invalid bucketId.",
				"com.remix.matchmaking",
				16102,
				http.StatusBadRequest,
			)
		},
		InvalidPartyPlayers: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.matchmaking.invalidPartyPlayers",
				"Blank or invalid partyPlayerIds.",
				"com.remix.matchmaking",
				16103,
				http.StatusBadRequest,
			)
		},
		InvalidPlatform: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.matchmaking.invalidPlatform",
				"Invalid platform.",
				"com.remix.matchmaking",
				16104,
				http.StatusBadRequest,
			)
		},
		NotAllowedIngame: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.matchmaking.notAllowedIngame",
				"Player is not allowed to play in game due to equipping items they do not own.",
				"com.remix.matchmaking",
				16105,
				http.StatusBadRequest,
			)
		},
	}

	Friends = struct {
		SelfFriend         func() *ApiError
		AccountNotFound    func() *ApiError
		FriendshipNotFound func() *ApiError
		RequestAlreadySent func() *ApiError
		InvalidData        func() *ApiError
	}{
		SelfFriend: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.friends.selfFriend",
				"You cannot be friend with yourself.",
				"com.remix.friends",
				14001,
				http.StatusBadRequest,
			)
		},
		AccountNotFound: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.friends.accountNotFound",
				"Account does not exist.",
				"com.remix.friends",
				14011,
				http.StatusNotFound,
			)
		},
		FriendshipNotFound: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.friends.friendshipNotFound",
				"Friendship does not exist.",
				"com.remix.friends",
				14004,
				http.StatusNotFound,
			)
		},
		RequestAlreadySent: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.friends.friendshipNotFound",
				"Friendship does not exist.",
				"com.remix.friends",
				14004,
				http.StatusNotFound,
			)
		},
		InvalidData: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.friends.invalidData",
				"Invalid data.",
				"com.remix.friends",
				14015,
				http.StatusBadRequest,
			)
		},
	}

	Internal = struct {
		ValidationFailed     func() *ApiError
		UnknownRoute         func() *ApiError
		InvalidUserAgent     func() *ApiError
		ServerError          func() *ApiError
		JsonParsingFailed    func() *ApiError
		RequestTimedOut      func() *ApiError
		UnsupportedMediaType func() *ApiError
		NotImplemented       func() *ApiError
		DataBaseError        func() *ApiError
		UnknownError         func() *ApiError
		EosError             func() *ApiError
	}{
		ValidationFailed: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.internal.validationFailed",
				"Validation Failed. Invalid fields were {0}.",
				"com.remix.internal",
				1040,
				http.StatusBadRequest,
			)
		},
		UnknownRoute: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.common.not_found",
				"Sorry, the resource you were trying to find could not be found.",
				"com.remix.internal",
				1004,
				http.StatusNotFound,
			)
		},
		InvalidUserAgent: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.internal.invalidUserAgent",
				"The user-agent header you provided does not match a Unreal Engine formatted user-agent.",
				"com.remix.internal",
				16183,
				http.StatusBadRequest,
			)
		},
		ServerError: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.internal.serverError",
				"Sorry, an error occurred and we were unable to resolve it.",
				"com.remix.internal",
				1000,
				http.StatusInternalServerError,
			)
		},
		JsonParsingFailed: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.internal.jsonParsingFailed",
				"JSON parse failed.",
				"com.remix.internal",
				1020,
				http.StatusBadRequest,
			)
		},
		RequestTimedOut: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.internal.requestTimedOut",
				"Request timed out.",
				"com.remix.internal",
				1001,
				http.StatusRequestTimeout,
			)
		},
		UnsupportedMediaType: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.internal.unsupportedMediaType",
				"Sorry, your request could not be processed because you provided a type of media that we do not support.",
				"com.remix.internal",
				1006,
				http.StatusUnsupportedMediaType,
			)
		},
		NotImplemented: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.internal.notImplemented",
				"The resource you were trying to access is not yet implemented by the server.",
				"com.remix.internal",
				1001,
				http.StatusNotImplemented,
			)
		},
		DataBaseError: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.internal.dataBaseError",
				"There was an error while interacting with the odin. Please report this issue.",
				"com.remix.internal",
				1001,
				http.StatusInternalServerError,
			)
		},
		UnknownError: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.internal.unknownError",
				"Sorry, an error occurred and we were unable to resolve it.",
				"com.remix.internal",
				1001,
				http.StatusInternalServerError,
			)
		},
		EosError: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.internal.EosError",
				"Sorry, an error occurred while communicating with Epic Online Service Servers.",
				"com.remix.internal",
				1001,
				http.StatusInternalServerError,
			)
		},
	}

	Basic = struct {
		BadRequest        func() *ApiError
		NotFound          func() *ApiError
		NotAcceptable     func() *ApiError
		MethodNotAllowed  func() *ApiError
		JsonMappingFailed func() *ApiError
		Throttled         func() *ApiError
	}{
		BadRequest: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.basic.badRequest",
				"Sorry, but your request is invalid.",
				"com.remix.basic",
				1001,
				http.StatusBadRequest,
			)
		},
		NotFound: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.basic.notFound",
				"The resource you were trying to find could not be found.",
				"com.remix.basic",
				1004,
				http.StatusNotFound,
			)
		},
		NotAcceptable: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.basic.notAcceptable",
				"Sorry, your request could not be processed as you do not accept the response type generated by this resource. Please check your Accept header.",
				"com.remix.basic",
				1008,
				http.StatusNotAcceptable,
			)
		},
		MethodNotAllowed: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.basic.methodNotAllowed",
				"Sorry, the resource you were trying to access cannot be accessed with the HTTP method you used.",
				"com.remix.basic",
				1009,
				http.StatusMethodNotAllowed,
			)
		},
		JsonMappingFailed: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.basic.jsonMappingFailed",
				"JSON mapping failed.",
				"com.remix.basic",
				1019,
				http.StatusBadRequest,
			)
		},
		Throttled: func() *ApiError {
			return NewApiError(
				"errors.com.epicgames.basic.throttled",
				"Operation access is limited by throttling policy.",
				"com.remix.basic",
				1041,
				http.StatusTooManyRequests,
			)
		},
	}
)
