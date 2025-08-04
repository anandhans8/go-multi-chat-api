package common

import (
	"go-multi-chat-api/src/infrastructure/helper"
	"math/rand"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type CommonService interface {
	GenerateRandomInteger() string
	AppendValidationErrors(ctx *gin.Context, ve validator.ValidationErrors, intr interface{})
}

type commonService struct {
	validator helper.Validator
}

func NewCommonService(validator helper.Validator) CommonService {
	return &commonService{
		validator: validator,
	}
}

const (
	min = 1000
	max = 9999
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (service *commonService) GenerateRandomInteger() string {
	otp := rand.Intn(max-min+1) + min
	i := strconv.Itoa(otp)

	return i
}

func (service *commonService) AppendValidationErrors(ctx *gin.Context, ve validator.ValidationErrors, intr interface{}) {
	type ErrorMsg struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	}
	out := make([]ErrorMsg, len(ve))

	for i, fe := range ve {
		//fieldName := fe.Namespace()
		Name, _ := jsonTag(intr, fe.Field())
		out[i] = ErrorMsg{Name, service.validator.GetErrorMsg(fe)}
	}
	ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": out})
	return

}

func jsonTag(v interface{}, fieldName string) (string, bool) {
	t := reflect.TypeOf(v)
	sf, ok := t.FieldByName(fieldName)
	if !ok {
		return "", false
	}
	return sf.Tag.Lookup("json")
}
