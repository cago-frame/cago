package validator

import (
	"errors"
	"reflect"
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
)

var trans ut.Translator
var uni *ut.UniversalTranslator

type DefaultValidator struct {
	validate *validator.Validate
}

func NewValidator() (binding.StructValidator, error) {
	ret := &DefaultValidator{
		validate: validator.New(),
	}
	ret.validate.SetTagName("binding")
	zhTran := zh.New()
	enTran := en.New()
	uni = ut.New(enTran, zhTran, enTran)
	var ok bool
	trans, ok = uni.GetTranslator("zh")
	if !ok {
		return nil, errors.New("uni.GetTranslator(\"zh\") failed")
	}
	err := zhTranslations.RegisterDefaultTranslations(ret.validate, trans)
	if err != nil {
		return nil, err
	}
	ret.registerValidation()
	ret.validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get("label")
		if name == "" {
			return fld.Name
		}
		return name
	})
	return ret, nil
}

func (v *DefaultValidator) registerValidation() {
	_ = v.validate.RegisterValidation("mobile", func(fl validator.FieldLevel) bool {
		if val, ok := fl.Field().Interface().(string); ok {
			reg := regexp.MustCompile("^1[0-9]{10}$")
			return reg.MatchString(val)
		}
		return false
	})
	_ = v.validate.RegisterTranslation("mobile", trans, func(ut ut.Translator) error {
		err := ut.Add("mobile", "手机号码格式不正确", false)
		return err
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("mobile")
		return t
	})

	_ = v.validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		if pwd, ok := fl.Field().Interface().(string); ok {
			if pwd == "" {
				return true
			}
			if len(pwd) < 8 || len(pwd) > 16 {
				return false
			}
			var level = 0
			patternList := []string{`[0-9]+`, `[a-z]+`, `[A-Z]+`, `[~!@#$%^&*?_-]+`}
			for _, pattern := range patternList {
				match, _ := regexp.MatchString(pattern, pwd)
				if match {
					level++
				}
			}
			return level >= 2
		}
		return false
	})
	_ = v.validate.RegisterTranslation("password", trans, func(ut ut.Translator) error {
		err := ut.Add("password", "密码不符合要求,必须包含数字,大小写字符,特殊字符其中的两种,长度不能小于8位大于16位", false)
		return err
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("password")
		return t
	})

	_ = v.validate.RegisterValidation("ignore", func(fl validator.FieldLevel) bool {
		if val, ok := fl.Field().Interface().(string); ok && val == "" {
			return false
		}
		return false
	})

}

func (v *DefaultValidator) ValidateStruct(obj interface{}) error {
	if kindOfData(obj) == reflect.Struct {
		if err := v.validate.Struct(obj); err != nil {
			return err
		}
	}
	return nil
}

func (v *DefaultValidator) Engine() interface{} {
	return v.validate
}

func kindOfData(data interface{}) reflect.Kind {

	value := reflect.ValueOf(data)
	valueType := value.Kind()

	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	return valueType
}

func TransError(e validator.ValidationErrors) string {
	return e[0].Translate(trans)
}
