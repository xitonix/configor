package configor

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	yaml "gopkg.in/yaml.v2"
)

// UnmatchedTomlKeysError errors are returned by the Load function when
// ErrorOnUnmatchedKeys is set to true and there are unmatched keys in the input
// toml config file. The string returned by Error() contains the names of the
// missing keys.
type UnmatchedTomlKeysError struct {
	Keys []toml.Key
}

func (e *UnmatchedTomlKeysError) Error() string {
	return fmt.Sprintf("There are keys in the config file that do not match any field in the given struct: %v", e.Keys)
}

func getConfigurationFileWithENVPrefix(file, env string) (string, error) {
	var (
		envFile string
		extname = path.Ext(file)
	)

	if extname == "" {
		envFile = fmt.Sprintf("%v.%v", file, env)
	} else {
		envFile = fmt.Sprintf("%v.%v%v", strings.TrimSuffix(file, extname), env, extname)
	}

	if fileInfo, err := os.Stat(envFile); err == nil && fileInfo.Mode().IsRegular() {
		return envFile, nil
	}
	return "", fmt.Errorf("failed to find file %v", file)
}

func (c *Configor) getConfigurationFiles(files ...string) []string {
	var results []string

	if c.Config.Debug || c.Config.Verbose {
		fmt.Printf("Current environment: '%v'\n", c.GetEnvironment())
	}

	for i := len(files) - 1; i >= 0; i-- {
		foundFile := false
		file := files[i]

		// check configuration
		if fileInfo, err := os.Stat(file); err == nil && fileInfo.Mode().IsRegular() {
			foundFile = true
			results = append(results, file)
		}

		// check configuration with env
		if file, err := getConfigurationFileWithENVPrefix(file, c.GetEnvironment()); err == nil {
			foundFile = true
			results = append(results, file)
		}

		// check example configuration
		if !foundFile {
			if example, err := getConfigurationFileWithENVPrefix(file, "example"); err == nil {
				fmt.Printf("Failed to find configuration %v, using example file %v\n", file, example)
				results = append(results, example)
			} else {
				fmt.Printf("Failed to find configuration %v\n", file)
			}
		}
	}
	return results
}

func processFile(config interface{}, file string, errorOnUnmatchedKeys bool) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	switch {
	case strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml"):
		if errorOnUnmatchedKeys {
			return yaml.UnmarshalStrict(data, config)
		}
		return yaml.Unmarshal(data, config)
	case strings.HasSuffix(file, ".toml"):
		return unmarshalToml(data, config, errorOnUnmatchedKeys)
	case strings.HasSuffix(file, ".json"):
		return unmarshalJSON(data, config, errorOnUnmatchedKeys)
	default:

		if err := unmarshalToml(data, config, errorOnUnmatchedKeys); err == nil {
			return nil
		} else if errUnmatchedKeys, ok := err.(*UnmatchedTomlKeysError); ok {
			return errUnmatchedKeys
		}

		if err := unmarshalJSON(data, config, errorOnUnmatchedKeys); err == nil {
			return nil
		} else if strings.Contains(err.Error(), "json: unknown field") {
			return err
		}

		var yamlError error
		if errorOnUnmatchedKeys {
			yamlError = yaml.UnmarshalStrict(data, config)
		} else {
			yamlError = yaml.Unmarshal(data, config)
		}

		if yamlError == nil {
			return nil
		} else if yErr, ok := yamlError.(*yaml.TypeError); ok {
			return yErr
		}

		return errors.New("failed to decode config")
	}
}

// GetStringTomlKeys returns a string array of the names of the keys that are passed in as args
func GetStringTomlKeys(list []toml.Key) []string {
	arr := make([]string, len(list))

	for index, key := range list {
		arr[index] = key.String()
	}
	return arr
}

func unmarshalToml(data []byte, config interface{}, errorOnUnmatchedKeys bool) error {
	metadata, err := toml.Decode(string(data), config)
	if err == nil && len(metadata.Undecoded()) > 0 && errorOnUnmatchedKeys {
		return &UnmatchedTomlKeysError{Keys: metadata.Undecoded()}
	}
	return err
}

func getPrefixForStruct(prefixes []string, fieldStruct *reflect.StructField) []string {
	if fieldStruct.Anonymous && fieldStruct.Tag.Get("anonymous") == "true" {
		return prefixes
	}
	result := make([]string, 0)
	for _, p := range prefixes {
		result = append(result, p+"_"+fieldStruct.Name)
	}

	jsonName := getJsonTag(fieldStruct)
	if jsonName != "" {
		for _, p := range prefixes {
			result = append(result, p+"_"+jsonName)
		}
	}

	if len(result) == 0 {
		result = append(result, fieldStruct.Name)
		if jsonName != "" {
			result = append(result, jsonName)
		}
	}

	return result
}

func getJsonTag(fieldStruct *reflect.StructField) string {
	tag := fieldStruct.Tag.Get("json")
	if len(tag) > 0 && tag != "-" {
		value := strings.TrimSpace(strings.TrimSpace(strings.Split(tag, ",")[0]))
		if len(value) > 0 && value != "-" {
			return value
		}
	}
	return ""
}

func (c *Configor) getEnvironmentVariables(fieldStruct reflect.StructField, prefixes ...string) []string {
	envTagValue := fieldStruct.Tag.Get("env")
	jsonTagValue := getJsonTag(&fieldStruct)

	if envTagValue != "" {
		result := []string{envTagValue}
		if len(c.globalPrefix) > 0 {
			result = append(result, c.globalPrefix+"_"+envTagValue, strings.ToUpper(c.globalPrefix)+"_"+envTagValue)
		}
		return result
	}

	result := make([]string, 0)

	for _, prefix := range prefixes {
		name := prefix + "_" + fieldStruct.Name
		result = append(result, name, strings.ToUpper(name))
		if len(jsonTagValue) > 0 {
			name = prefix + "_" + jsonTagValue
			result = append(result, name, strings.ToUpper(name))
		}
	}

	if len(result) == 0 {
		result = []string{fieldStruct.Name, strings.ToUpper(fieldStruct.Name)}
		if len(jsonTagValue) > 0 {
			result = append(result, jsonTagValue, strings.ToUpper(jsonTagValue))
		}
	}

	return result
}

func (c *Configor) processTags(config interface{}, prefixes ...string) error {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	if configValue.Kind() != reflect.Struct {
		return errors.New("invalid config, should be struct")
	}

	configType := configValue.Type()
	for i := 0; i < configType.NumField(); i++ {
		var (
			fieldStruct = configType.Field(i)
			field       = configValue.Field(i)
		)

		if field.Kind() == reflect.Ptr && field.IsNil() {
			// Nested pointers with nil value
			field = reflect.New(field.Type().Elem()).Elem()
		}

		if !field.CanAddr() || !field.CanInterface() {
			continue
		}

		envNames := c.getEnvironmentVariables(fieldStruct, prefixes...)

		if c.Config.Verbose {
			fmt.Printf("Trying to load struct `%v`'s field `%v` from env %v\n", configType.Name(), fieldStruct.Name, strings.Join(envNames, ", "))
		}

		// Load From Shell ENV
		for _, env := range envNames {
			if value := os.Getenv(env); value != "" {
				if c.Config.Debug || c.Config.Verbose {
					fmt.Printf("Loading configuration for struct `%v`'s field `%v` from env %v...\n", configType.Name(), fieldStruct.Name, env)
				}
				if err := yaml.Unmarshal([]byte(value), field.Addr().Interface()); err != nil {
					return err
				}
				break
			}
		}

		if isBlank := reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()); isBlank {
			// Set default configuration if blank
			if value := fieldStruct.Tag.Get("default"); value != "" {
				if err := yaml.Unmarshal([]byte(value), field.Addr().Interface()); err != nil {
					return err
				}
			} else if fieldStruct.Tag.Get("required") == "true" {
				// return error if it is required but blank
				name := fieldStruct.Name
				if len(envNames) > 0 {
					name = strings.ToUpper(envNames[len(envNames)-1])
				}
				return errors.New(name + " is required, but blank")
			}
		}

		for field.Kind() == reflect.Ptr {
			field = field.Elem()
		}

		if field.Kind() == reflect.Struct {
			if err := c.processTags(field.Addr().Interface(), getPrefixForStruct(prefixes, &fieldStruct)...); err != nil {
				return err
			}
		}

		if field.Kind() == reflect.Slice {
			for i := 0; i < field.Len(); i++ {
				if reflect.Indirect(field.Index(i)).Kind() == reflect.Struct {
					if err := c.processTags(field.Index(i).Addr().Interface(), append(getPrefixForStruct(prefixes, &fieldStruct), fmt.Sprint(i))...); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
