package configor_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/BurntSushi/toml"
	"github.com/xitonix/configor"
)

type Anonymous struct {
	Description string
}

type Connection struct {
	Name     string
	User     string `json:"user_name" default:"root"`
	Password string `json:"pass" required:"true" env:"DBPassword"`
	Port     uint   `default:"3306"`
	Endpoint string `json:"ep,omitempty" required:"true"`
}

type Contact struct {
	Name     string `json:"first_name"`
	LastName string
	Email    string `required:"true"`
}

type Config struct {
	APPName string `default:"configor"`
	Hosts   []string

	DB Connection `required: true`

	Contacts       []Contact
	PrimaryContact Contact  `json:"primary_contact"`
	ContactPtr     *Contact `json:"contact_ptr"`

	Anonymous `anonymous:"true"`

	private string
}

func generateDefaultConfig() Config {
	config := Config{
		APPName: "configor",
		Hosts:   []string{"http://example.org", "http://xitonix.me"},
		DB: Connection{
			Name:     "configor",
			User:     "configor",
			Password: "configor",
			Port:     3306,
			Endpoint: "configor",
		},
		Contacts: []Contact{
			{
				Name:  "xitonix",
				Email: "wosmvp@gmail.com",
			},
		},
		PrimaryContact: Contact{
			Name:     "configor",
			LastName: "configor",
			Email:    "configor@xitonix.io",
		},
		ContactPtr: &Contact{
			Name:     "configor",
			LastName: "configor",
			Email:    "configor@xitonix.io",
		},
		Anonymous: Anonymous{
			Description: "This is an anonymous embedded struct whose environment variables should NOT include 'ANONYMOUS'",
		},
	}
	return config
}

func TestLoadNormalConfig(t *testing.T) {
	config := generateDefaultConfig()
	if bytes, err := json.Marshal(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(bytes)

			var result Config
			configor.Load(&result, file.Name())
			if !reflect.DeepEqual(result, config) {
				t.Errorf("result should equal to original configuration")
			}
		}
	} else {
		t.Errorf("failed to marshal config")
	}
}

func TestLoadConfigFromTomlWithExtension(t *testing.T) {
	var (
		config = generateDefaultConfig()
		buffer bytes.Buffer
	)

	if err := toml.NewEncoder(&buffer).Encode(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor.toml"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(buffer.Bytes())

			var result Config
			configor.Load(&result, file.Name())
			if !reflect.DeepEqual(result, config) {
				t.Errorf("result should equal to original configuration")
			}
		}
	} else {
		t.Errorf("failed to marshal config")
	}
}

func TestLoadConfigFromTomlWithoutExtension(t *testing.T) {
	var (
		config = generateDefaultConfig()
		buffer bytes.Buffer
	)

	if err := toml.NewEncoder(&buffer).Encode(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(buffer.Bytes())

			var result Config
			configor.Load(&result, file.Name())
			if !reflect.DeepEqual(result, config) {
				t.Errorf("result should equal to original configuration")
			}
		}
	} else {
		t.Errorf("failed to marshal config")
	}
}

func TestDefaultValue(t *testing.T) {
	config := generateDefaultConfig()
	config.APPName = ""
	config.DB.Port = 0

	if bytes, err := json.Marshal(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(bytes)

			var result Config
			configor.Load(&result, file.Name())
			if !reflect.DeepEqual(result, generateDefaultConfig()) {
				t.Errorf("result should be set default value correctly")
			}
		}
	} else {
		t.Errorf("failed to marshal config")
	}
}

func TestMissingRequiredValue(t *testing.T) {
	config := generateDefaultConfig()
	config.DB.Password = ""

	if bytes, err := json.Marshal(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(bytes)

			var result Config
			if err := configor.Load(&result, file.Name()); err == nil {
				t.Errorf("Should got error when load configuration missing db password")
			}
		}
	} else {
		t.Errorf("failed to marshal config")
	}
}

func TestUnmatchedKeyInTomlConfigFile(t *testing.T) {
	type configStruct struct {
		Name string
	}
	type configFile struct {
		Name string
		Test string
	}
	config := configFile{Name: "test", Test: "ATest"}

	file, err := ioutil.TempFile("/tmp", "configor")
	if err != nil {
		t.Fatal("Could not create temp file")
	}
	defer os.Remove(file.Name())
	defer file.Close()

	filename := file.Name()

	if err := toml.NewEncoder(file).Encode(config); err == nil {

		var result configStruct

		// Do not return error when there are unmatched keys but ErrorOnUnmatchedKeys is false
		if err := configor.New(&configor.Config{}).Load(&result, filename); err != nil {
			t.Errorf("Should NOT get error when loading configuration with extra keys")
		}

		// Return an error when there are unmatched keys and ErrorOnUnmatchedKeys is true
		err := configor.New(&configor.Config{ErrorOnUnmatchedKeys: true}).Load(&result, filename)
		if err == nil {
			t.Errorf("Should get error when loading configuration with extra keys")
		}

		// The error should be of type UnmatchedTomlKeysError
		tomlErr, ok := err.(*configor.UnmatchedTomlKeysError)
		if !ok {
			t.Errorf("Should get UnmatchedTomlKeysError error when loading configuration with extra keys")
		}

		// The error.Keys() function should return the "Test" key
		keys := configor.GetStringTomlKeys(tomlErr.Keys)
		if len(keys) != 1 || keys[0] != "Test" {
			t.Errorf("The UnmatchedTomlKeysError should contain the Test key")
		}

	} else {
		t.Errorf("failed to marshal config")
	}

	// Add .toml to the file name and test again
	err = os.Rename(filename, filename+".toml")
	if err != nil {
		t.Errorf("Could not add suffix to file")
	}
	filename = filename + ".toml"
	defer os.Remove(filename)

	var result configStruct

	// Do not return error when there are unmatched keys but ErrorOnUnmatchedKeys is false
	if err := configor.New(&configor.Config{}).Load(&result, filename); err != nil {
		t.Errorf("Should NOT get error when loading configuration with extra keys. Error: %v", err)
	}

	// Return an error when there are unmatched keys and ErrorOnUnmatchedKeys is true
	err = configor.New(&configor.Config{ErrorOnUnmatchedKeys: true}).Load(&result, filename)
	if err == nil {
		t.Errorf("Should get error when loading configuration with extra keys")
	}

	// The error should be of type UnmatchedTomlKeysError
	tomlErr, ok := err.(*configor.UnmatchedTomlKeysError)
	if !ok {
		t.Errorf("Should get UnmatchedTomlKeysError error when loading configuration with extra keys")
	}

	// The error.Keys() function should return the "Test" key
	keys := configor.GetStringTomlKeys(tomlErr.Keys)
	if len(keys) != 1 || keys[0] != "Test" {
		t.Errorf("The UnmatchedTomlKeysError should contain the Test key")
	}

}

func TestUnmatchedKeyInYamlConfigFile(t *testing.T) {
	type configStruct struct {
		Name string
	}
	type configFile struct {
		Name string
		Test string
	}
	config := configFile{Name: "test", Test: "ATest"}

	file, err := ioutil.TempFile("/tmp", "configor")
	if err != nil {
		t.Fatal("Could not create temp file")
	}

	defer os.Remove(file.Name())
	defer file.Close()

	filename := file.Name()

	if data, err := yaml.Marshal(config); err == nil {
		file.WriteString(string(data))

		var result configStruct

		// Do not return error when there are unmatched keys but ErrorOnUnmatchedKeys is false
		if err := configor.New(&configor.Config{}).Load(&result, filename); err != nil {
			t.Errorf("Should NOT get error when loading configuration with extra keys. Error: %v", err)
		}

		// Return an error when there are unmatched keys and ErrorOnUnmatchedKeys is true
		if err := configor.New(&configor.Config{ErrorOnUnmatchedKeys: true}).Load(&result, filename); err == nil {
			t.Errorf("Should get error when loading configuration with extra keys")

			// The error should be of type *yaml.TypeError
		} else if _, ok := err.(*yaml.TypeError); !ok {
			// || !strings.Contains(err.Error(), "not found in struct") {
			t.Errorf("Error should be of type yaml.TypeError. Instead error is %v", err)
		}

	} else {
		t.Errorf("failed to marshal config")
	}

	// Add .yaml to the file name and test again
	err = os.Rename(filename, filename+".yaml")
	if err != nil {
		t.Errorf("Could not add suffix to file")
	}
	filename = filename + ".yaml"
	defer os.Remove(filename)

	var result configStruct

	// Do not return error when there are unmatched keys but ErrorOnUnmatchedKeys is false
	if err := configor.New(&configor.Config{}).Load(&result, filename); err != nil {
		t.Errorf("Should NOT get error when loading configuration with extra keys. Error: %v", err)
	}

	// Return an error when there are unmatched keys and ErrorOnUnmatchedKeys is true
	if err := configor.New(&configor.Config{ErrorOnUnmatchedKeys: true}).Load(&result, filename); err == nil {
		t.Errorf("Should get error when loading configuration with extra keys")

		// The error should be of type *yaml.TypeError
	} else if _, ok := err.(*yaml.TypeError); !ok {
		// || !strings.Contains(err.Error(), "not found in struct") {
		t.Errorf("Error should be of type yaml.TypeError. Instead error is %v", err)
	}
}

func TestLoadConfigurationByEnvironment(t *testing.T) {
	config := generateDefaultConfig()
	config2 := struct {
		APPName string
	}{
		APPName: "config2",
	}

	if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
		defer file.Close()
		defer os.Remove(file.Name())
		configBytes, _ := yaml.Marshal(config)
		config2Bytes, _ := yaml.Marshal(config2)
		ioutil.WriteFile(file.Name()+".yaml", configBytes, 0644)
		defer os.Remove(file.Name() + ".yaml")
		ioutil.WriteFile(file.Name()+".production.yaml", config2Bytes, 0644)
		defer os.Remove(file.Name() + ".production.yaml")

		var result Config
		os.Setenv("CONFIGOR_ENV", "production")
		defer os.Setenv("CONFIGOR_ENV", "")
		if err := configor.Load(&result, file.Name()+".yaml"); err != nil {
			t.Errorf("No error should happen when load configurations, but got %v", err)
		}

		var defaultConfig = generateDefaultConfig()
		defaultConfig.APPName = "config2"
		if !reflect.DeepEqual(result, defaultConfig) {
			t.Errorf("result should be load configurations by environment correctly")
		}
	}
}

func TestLoadConfigurationByEnvironmentSetByConfig(t *testing.T) {
	config := generateDefaultConfig()
	config2 := struct {
		APPName string
	}{
		APPName: "production_config2",
	}

	if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
		defer file.Close()
		defer os.Remove(file.Name())
		configBytes, _ := yaml.Marshal(config)
		config2Bytes, _ := yaml.Marshal(config2)
		ioutil.WriteFile(file.Name()+".yaml", configBytes, 0644)
		defer os.Remove(file.Name() + ".yaml")
		ioutil.WriteFile(file.Name()+".production.yaml", config2Bytes, 0644)
		defer os.Remove(file.Name() + ".production.yaml")

		var result Config
		var Configor = configor.New(&configor.Config{Environment: "production"})
		if Configor.Load(&result, file.Name()+".yaml"); err != nil {
			t.Errorf("No error should happen when load configurations, but got %v", err)
		}

		var defaultConfig = generateDefaultConfig()
		defaultConfig.APPName = "production_config2"
		if !reflect.DeepEqual(result, defaultConfig) {
			t.Errorf("result should be load configurations by environment correctly")
		}

		if Configor.GetEnvironment() != "production" {
			t.Errorf("configor's environment should be production")
		}
	}
}

func TestOverwriteConfigurationWithEnvironmentWithDefaultPrefix(t *testing.T) {
	config := generateDefaultConfig()

	if bytes, err := json.Marshal(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(bytes)
			var result Config
			os.Setenv("CONFIGOR_APPNAME", "config2")
			os.Setenv("CONFIGOR_HOSTS", "- http://example.org\n- http://xitonix.me")
			os.Setenv("CONFIGOR_DB_NAME", "db_name")
			defer os.Setenv("CONFIGOR_APPNAME", "")
			defer os.Setenv("CONFIGOR_HOSTS", "")
			defer os.Setenv("CONFIGOR_DB_NAME", "")
			configor.Load(&result, file.Name())

			var defaultConfig = generateDefaultConfig()
			defaultConfig.APPName = "config2"
			defaultConfig.Hosts = []string{"http://example.org", "http://xitonix.me"}
			defaultConfig.DB.Name = "db_name"
			if !reflect.DeepEqual(result, defaultConfig) {
				t.Errorf("result should equal to original configuration")
			}
		}
	}
}

func TestOverwriteConfigurationWithEnvironment(t *testing.T) {
	config := generateDefaultConfig()

	if bytes, err := json.Marshal(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(bytes)
			var result Config
			os.Setenv("CONFIGOR_ENV_PREFIX", "app")
			os.Setenv("APP_APPNAME", "config2")
			os.Setenv("APP_DB_NAME", "db_name")
			defer os.Setenv("CONFIGOR_ENV_PREFIX", "")
			defer os.Setenv("APP_APPNAME", "")
			defer os.Setenv("APP_DB_NAME", "")
			configor.Load(&result, file.Name())

			var defaultConfig = generateDefaultConfig()
			defaultConfig.APPName = "config2"
			defaultConfig.DB.Name = "db_name"
			if !reflect.DeepEqual(result, defaultConfig) {
				t.Errorf("result should equal to original configuration")
			}
		}
	}
}

func TestOverwriteConfigurationWithJsonTag(t *testing.T) {
	config := generateDefaultConfig()

	bytes, err := json.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal the configuration object: %s", err)
	}

	file, err := ioutil.TempFile("/tmp", "configor")
	if err != nil {
		t.Errorf("Failed to create the temp file: %s", err)
	}

	_, err = file.Write(bytes)
	if err != nil {
		t.Errorf("Failed write into the temp file: %s", err)
	}

	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()

	testCases := []struct {
		title            string
		withGlobalPrefix bool
		usernameEnvTag   string
		expectedUserName string
		passwordEnvTag   string
		expectedPassword string
		endpointEnvTag   string
		expectedEndpoint string
	}{
		{
			title:            "with global prefix and with json tags environment variables",
			withGlobalPrefix: true,

			usernameEnvTag:   "DB_USER_NAME",
			expectedUserName: "env user name",

			passwordEnvTag:   "DBPassword",
			expectedPassword: "env password",

			endpointEnvTag:   "DB_EP",
			expectedEndpoint: "env endpoint",
		},
		{
			title:            "with global prefix and with field name environment variables",
			withGlobalPrefix: true,

			usernameEnvTag:   "DB_USER",
			expectedUserName: "env user name",

			passwordEnvTag:   "DBPassword",
			expectedPassword: "env password",

			endpointEnvTag:   "DB_ENDPOINT",
			expectedEndpoint: "env endpoint",
		},
		{
			title:            "with global prefix and with json tags environment variables when there is an env struct tag override",
			withGlobalPrefix: true,

			usernameEnvTag:   "DB_USER_NAME",
			expectedUserName: "env user name",

			passwordEnvTag:   "DB_PASS",
			expectedPassword: "configor",

			endpointEnvTag:   "DB_EP",
			expectedEndpoint: "env endpoint",
		},

		{
			title:            "without global prefix and with json tags environment variables",
			withGlobalPrefix: false,

			usernameEnvTag:   "DB_USER_NAME",
			expectedUserName: "env user name",

			passwordEnvTag:   "DBPassword",
			expectedPassword: "env password",

			endpointEnvTag:   "DB_EP",
			expectedEndpoint: "env endpoint",
		},
		{
			title:            "without global prefix and with field name environment variables",
			withGlobalPrefix: false,

			usernameEnvTag:   "DB_USER",
			expectedUserName: "env user name",

			passwordEnvTag:   "DBPassword",
			expectedPassword: "env password",

			endpointEnvTag:   "DB_ENDPOINT",
			expectedEndpoint: "env endpoint",
		},
		{
			title:            "without global prefix and with json tags environment variables when there is an `env` struct tag override",
			withGlobalPrefix: false,

			usernameEnvTag:   "DB_USER_NAME",
			expectedUserName: "env user name",

			passwordEnvTag:   "DB_PASS",
			expectedPassword: "configor",

			endpointEnvTag:   "DB_EP",
			expectedEndpoint: "env endpoint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {

			var prefix string
			if tc.withGlobalPrefix {
				_ = os.Setenv("CONFIGOR_ENV_PREFIX", "app")
				prefix = "APP_"
			} else {
				_ = os.Setenv("CONFIGOR_ENV_PREFIX", "-")
			}

			_ = os.Setenv(prefix+tc.usernameEnvTag, tc.expectedUserName)
			_ = os.Setenv(prefix+tc.passwordEnvTag, tc.expectedPassword)
			_ = os.Setenv(prefix+tc.endpointEnvTag, tc.expectedEndpoint)

			defer func() {
				_ = os.Setenv(prefix+tc.usernameEnvTag, "")
				_ = os.Setenv(prefix+tc.passwordEnvTag, "")
				_ = os.Setenv(prefix+tc.endpointEnvTag, "")
				_ = os.Setenv("CONFIGOR_ENV_PREFIX", "")
			}()

			var result Config
			err = configor.Load(&result, file.Name())
			if err != nil {
				t.Errorf("failed to load the temp config file: %s", err)
			}

			if result.DB.User != tc.expectedUserName {
				t.Errorf("DB User | Expected: %s, Actual: %s", tc.expectedUserName, result.DB.User)
			}

			if result.DB.Password != tc.expectedPassword {
				t.Errorf("DB Password | Expected: %s, Actual: %s", tc.expectedPassword, result.DB.Password)
			}

			if result.DB.Endpoint != tc.expectedEndpoint {
				t.Errorf("DB Endpoint | Expected: %s, Actual: %s", tc.expectedEndpoint, result.DB.Endpoint)
			}
		})
	}
}

func TestOverwriteConfigurationOfNestedTypeWithJsonTag(t *testing.T) {
	config := generateDefaultConfig()

	bytes, err := json.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal the configuration object: %s", err)
	}

	file, err := ioutil.TempFile("/tmp", "configor")
	if err != nil {
		t.Errorf("Failed to create the temp file: %s", err)
	}

	_, err = file.Write(bytes)
	if err != nil {
		t.Errorf("Failed write into the temp file: %s", err)
	}

	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()

	testCases := []struct {
		title             string
		withGlobalPrefix  bool
		firstNameEnvTag   string
		expectedFirstName string
		lastNameEnvTag    string
		expectedLastName  string
	}{
		{
			title:            "with global prefix and with json tags environment variables",
			withGlobalPrefix: true,

			firstNameEnvTag:   "PRIMARY_CONTACT_FIRST_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "PRIMARY_CONTACT_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "with global prefix and field name",
			withGlobalPrefix: true,

			firstNameEnvTag:   "PRIMARYCONTACT_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "PRIMARYCONTACT_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "with global prefix and with parent json tags environment variables",
			withGlobalPrefix: true,

			firstNameEnvTag:   "PRIMARY_CONTACT_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "PRIMARY_CONTACT_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "with global prefix parent filed name and child json tag environment variables",
			withGlobalPrefix: true,

			firstNameEnvTag:   "PRIMARYCONTACT_FIRST_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "PRIMARYCONTACT_LASTNAME",
			expectedLastName: "env last name",
		},

		{
			title:            "without global prefix and with json tags environment variables",
			withGlobalPrefix: false,

			firstNameEnvTag:   "PRIMARY_CONTACT_FIRST_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "PRIMARY_CONTACT_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "without global prefix and field name",
			withGlobalPrefix: false,

			firstNameEnvTag:   "PRIMARYCONTACT_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "PRIMARYCONTACT_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "without global prefix and with parent json tags environment variables",
			withGlobalPrefix: false,

			firstNameEnvTag:   "PRIMARY_CONTACT_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "PRIMARY_CONTACT_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "without global prefix parent filed name and child json tag environment variables",
			withGlobalPrefix: false,

			firstNameEnvTag:   "PRIMARYCONTACT_FIRST_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "PRIMARYCONTACT_LASTNAME",
			expectedLastName: "env last name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {

			var prefix string
			if tc.withGlobalPrefix {
				_ = os.Setenv("CONFIGOR_ENV_PREFIX", "app")
				prefix = "APP_"
			} else {
				_ = os.Setenv("CONFIGOR_ENV_PREFIX", "-")
			}

			_ = os.Setenv(prefix+tc.firstNameEnvTag, tc.expectedFirstName)
			_ = os.Setenv(prefix+tc.lastNameEnvTag, tc.expectedLastName)

			defer func() {
				_ = os.Setenv(prefix+tc.firstNameEnvTag, "")
				_ = os.Setenv(prefix+tc.lastNameEnvTag, "")
				_ = os.Setenv("CONFIGOR_ENV_PREFIX", "")
			}()

			var result Config
			err = configor.Load(&result, file.Name())
			if err != nil {
				t.Errorf("failed to load the temp config file: %s", err)
			}

			if result.PrimaryContact.Name != tc.expectedFirstName {
				t.Errorf("First Name | Expected: %s, Actual: %s", tc.expectedFirstName, result.PrimaryContact.Name)
			}

			if result.PrimaryContact.LastName != tc.expectedLastName {
				t.Errorf("Last Name | Expected: %s, Actual: %s", tc.expectedLastName, result.PrimaryContact.LastName)
			}
		})
	}
}

func TestOverwriteConfigurationOfNestedPointerWithJsonTag(t *testing.T) {
	config := generateDefaultConfig()

	bytes, err := json.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal the configuration object: %s", err)
	}

	file, err := ioutil.TempFile("/tmp", "configor")
	if err != nil {
		t.Errorf("Failed to create the temp file: %s", err)
	}

	_, err = file.Write(bytes)
	if err != nil {
		t.Errorf("Failed write into the temp file: %s", err)
	}

	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()

	testCases := []struct {
		title             string
		withGlobalPrefix  bool
		firstNameEnvTag   string
		expectedFirstName string
		lastNameEnvTag    string
		expectedLastName  string
	}{
		{
			title:            "with global prefix and with json tags environment variables",
			withGlobalPrefix: true,

			firstNameEnvTag:   "CONTACT_PTR_FIRST_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "CONTACT_PTR_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "with global prefix and field name",
			withGlobalPrefix: true,

			firstNameEnvTag:   "CONTACTPTR_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "CONTACTPTR_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "with global prefix and with parent json tags environment variables",
			withGlobalPrefix: true,

			firstNameEnvTag:   "CONTACT_PTR_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "CONTACT_PTR_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "with global prefix parent filed name and child json tag environment variables",
			withGlobalPrefix: true,

			firstNameEnvTag:   "CONTACTPTR_FIRST_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "CONTACTPTR_LASTNAME",
			expectedLastName: "env last name",
		},

		{
			title:            "without global prefix and with json tags environment variables",
			withGlobalPrefix: false,

			firstNameEnvTag:   "CONTACT_PTR_FIRST_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "CONTACT_PTR_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "without global prefix and field name",
			withGlobalPrefix: false,

			firstNameEnvTag:   "CONTACTPTR_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "CONTACTPTR_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "without global prefix and with parent json tags environment variables",
			withGlobalPrefix: false,

			firstNameEnvTag:   "CONTACT_PTR_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "CONTACT_PTR_LASTNAME",
			expectedLastName: "env last name",
		},
		{
			title:            "without global prefix parent filed name and child json tag environment variables",
			withGlobalPrefix: false,

			firstNameEnvTag:   "CONTACTPTR_FIRST_NAME",
			expectedFirstName: "env first name",

			lastNameEnvTag:   "CONTACTPTR_LASTNAME",
			expectedLastName: "env last name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {

			var prefix string
			if tc.withGlobalPrefix {
				_ = os.Setenv("CONFIGOR_ENV_PREFIX", "app")
				prefix = "APP_"
			} else {
				_ = os.Setenv("CONFIGOR_ENV_PREFIX", "-")
			}

			_ = os.Setenv(prefix+tc.firstNameEnvTag, tc.expectedFirstName)
			_ = os.Setenv(prefix+tc.lastNameEnvTag, tc.expectedLastName)

			defer func() {
				_ = os.Setenv(prefix+tc.firstNameEnvTag, "")
				_ = os.Setenv(prefix+tc.lastNameEnvTag, "")
				_ = os.Setenv("CONFIGOR_ENV_PREFIX", "")
			}()

			var result Config
			err = configor.Load(&result, file.Name())
			if err != nil {
				t.Errorf("failed to load the temp config file: %s", err)
			}

			if result.ContactPtr.Name != tc.expectedFirstName {
				t.Errorf("First Name | Expected: %s, Actual: %s", tc.expectedFirstName, result.ContactPtr.Name)
			}

			if result.ContactPtr.LastName != tc.expectedLastName {
				t.Errorf("Last Name | Expected: %s, Actual: %s", tc.expectedLastName, result.ContactPtr.LastName)
			}
		})
	}
}

func TestOverwriteConfigurationWithEnvironmentThatSetByConfig(t *testing.T) {
	config := generateDefaultConfig()

	if bytes, err := json.Marshal(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(bytes)
			os.Setenv("APP1_APPName", "config2")
			os.Setenv("APP1_DB_Name", "db_name")
			defer os.Setenv("APP1_APPName", "")
			defer os.Setenv("APP1_DB_Name", "")

			var result Config
			var Configor = configor.New(&configor.Config{ENVPrefix: "APP1"})
			Configor.Load(&result, file.Name())

			var defaultConfig = generateDefaultConfig()
			defaultConfig.APPName = "config2"
			defaultConfig.DB.Name = "db_name"
			if !reflect.DeepEqual(result, defaultConfig) {
				t.Errorf("result should equal to original configuration")
			}
		}
	}
}

func TestResetPrefixToBlank(t *testing.T) {
	config := generateDefaultConfig()

	if bytes, err := json.Marshal(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(bytes)
			var result Config
			os.Setenv("CONFIGOR_ENV_PREFIX", "-")
			os.Setenv("APPNAME", "config2")
			os.Setenv("DB_NAME", "db_name")
			defer os.Setenv("CONFIGOR_ENV_PREFIX", "")
			defer os.Setenv("APPNAME", "")
			defer os.Setenv("DB_NAME", "")

			configor.Load(&result, file.Name())

			var defaultConfig = generateDefaultConfig()
			defaultConfig.APPName = "config2"
			defaultConfig.DB.Name = "db_name"
			if !reflect.DeepEqual(result, defaultConfig) {
				t.Errorf("result should equal to original configuration")
			}
		}
	}
}

func TestResetPrefixToBlank2(t *testing.T) {
	config := generateDefaultConfig()

	if bytes, err := json.Marshal(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(bytes)
			var result Config
			os.Setenv("CONFIGOR_ENV_PREFIX", "-")
			os.Setenv("APPName", "config2")
			os.Setenv("DB_Name", "db_name")
			defer os.Setenv("CONFIGOR_ENV_PREFIX", "")
			defer os.Setenv("APPName", "")
			defer os.Setenv("DB_Name", "")
			configor.Load(&result, file.Name())

			var defaultConfig = generateDefaultConfig()
			defaultConfig.APPName = "config2"
			defaultConfig.DB.Name = "db_name"
			if !reflect.DeepEqual(result, defaultConfig) {
				t.Errorf("result should equal to original configuration")
			}
		}
	}
}

func TestReadFromEnvironmentWithSpecifiedEnvName(t *testing.T) {
	config := generateDefaultConfig()

	if bytes, err := json.Marshal(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(bytes)
			var result Config
			os.Setenv("DBPassword", "db_password")
			defer os.Setenv("DBPassword", "")
			configor.Load(&result, file.Name())

			var defaultConfig = generateDefaultConfig()
			defaultConfig.DB.Password = "db_password"
			if !reflect.DeepEqual(result, defaultConfig) {
				t.Errorf("result should equal to original configuration")
			}
		}
	}
}

func TestAnonymousStruct(t *testing.T) {
	config := generateDefaultConfig()

	if bytes, err := json.Marshal(config); err == nil {
		if file, err := ioutil.TempFile("/tmp", "configor"); err == nil {
			defer file.Close()
			defer os.Remove(file.Name())
			file.Write(bytes)
			var result Config
			os.Setenv("CONFIGOR_DESCRIPTION", "environment description")
			defer os.Setenv("CONFIGOR_DESCRIPTION", "")
			configor.Load(&result, file.Name())

			var defaultConfig = generateDefaultConfig()
			defaultConfig.Anonymous.Description = "environment description"
			if !reflect.DeepEqual(result, defaultConfig) {
				t.Errorf("result should equal to original configuration")
			}
		}
	}
}

func TestENV(t *testing.T) {
	if configor.ENV() != "test" {
		t.Errorf("Env should be test when running `go test`, instead env is %v", configor.ENV())
	}

	os.Setenv("CONFIGOR_ENV", "production")
	defer os.Setenv("CONFIGOR_ENV", "")
	if configor.ENV() != "production" {
		t.Errorf("Env should be production when set it with CONFIGOR_ENV")
	}
}
