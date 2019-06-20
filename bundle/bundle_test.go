package bundle

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/deislabs/cnab-go/bundle/definition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadTopLevelProperties(t *testing.T) {
	json := `{
		"schemaVersion": "v1.0.0-WD",
		"name": "foo",
		"version": "1.0",
		"images": {},
		"credentials": {},
		"custom": {}
	}`
	bundle, err := Unmarshal([]byte(json))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "v1.0.0-WD", bundle.SchemaVersion)
	if bundle.Name != "foo" {
		t.Errorf("Expected name 'foo', got '%s'", bundle.Name)
	}
	if bundle.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", bundle.Version)
	}
	if len(bundle.Images) != 0 {
		t.Errorf("Expected no images, got %d", len(bundle.Images))
	}
	if len(bundle.Credentials) != 0 {
		t.Errorf("Expected no credentials, got %d", len(bundle.Credentials))
	}
	if len(bundle.Custom) != 0 {
		t.Errorf("Expected no custom extensions, got %d", len(bundle.Custom))
	}
}

func TestReadImageProperties(t *testing.T) {
	data, err := ioutil.ReadFile("../testdata/bundles/foo.json")
	if err != nil {
		t.Errorf("cannot read bundle file: %v", err)
	}

	bundle, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(bundle.Images))
	}
	image1 := bundle.Images["image1"]
	if image1.Description != "image1" {
		t.Errorf("Expected description 'image1', got '%s'", image1.Description)
	}
	if image1.Image != "urn:image1uri" {
		t.Errorf("Expected Image 'urn:image1uri', got '%s'", image1.Image)
	}
	if image1.OriginalImage != "urn:image1originaluri" {
		t.Errorf("Expected Image 'urn:image1originaluri', got '%s'", image1.OriginalImage)
	}
	image2 := bundle.Images["image2"]
	if image2.OriginalImage != "" {
		t.Errorf("Expected Image '', got '%s'", image2.OriginalImage)
	}
}

func TestReadCredentialProperties(t *testing.T) {
	data, err := ioutil.ReadFile("../testdata/bundles/foo.json")
	if err != nil {
		t.Errorf("cannot read bundle file: %v", err)
	}

	bundle, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Credentials) != 3 {
		t.Errorf("Expected 3 credentials, got %d", len(bundle.Credentials))
	}
	f := bundle.Credentials["foo"]
	if f.Path != "pfoo" {
		t.Errorf("Expected path 'pfoo', got '%s'", f.Path)
	}
	if f.EnvironmentVariable != "" {
		t.Errorf("Expected env '', got '%s'", f.EnvironmentVariable)
	}
	b := bundle.Credentials["bar"]
	if b.Path != "" {
		t.Errorf("Expected path '', got '%s'", b.Path)
	}
	if b.EnvironmentVariable != "ebar" {
		t.Errorf("Expected env 'ebar', got '%s'", b.EnvironmentVariable)
	}
	q := bundle.Credentials["quux"]
	if q.Path != "pquux" {
		t.Errorf("Expected path 'pquux', got '%s'", q.Path)
	}
	if q.EnvironmentVariable != "equux" {
		t.Errorf("Expected env 'equux', got '%s'", q.EnvironmentVariable)
	}
}

func TestValuesOrDefaults(t *testing.T) {
	is := assert.New(t)
	vals := map[string]interface{}{
		"port":    8080,
		"host":    "localhost",
		"enabled": true,
	}
	b := &Bundle{
		Definitions: map[string]*definition.Schema{
			"portType": {
				Type:    "integer",
				Default: 1234,
			},
			"hostType": {
				Type:    "string",
				Default: "locahost.localdomain",
			},
			"replicaCountType": {
				Type:    "integer",
				Default: 3,
			},
			"enabledType": {
				Type:    "boolean",
				Default: false,
			},
		},
		Parameters: ParametersDefinition{
			Fields: map[string]ParameterDefinition{
				"port": {
					Definition: "portType",
				},
				"host": {
					Definition: "hostType",
				},
				"enabled": {
					Definition: "enabledType",
				},
				"replicaCount": {
					Definition: "replicaCountType",
				},
			},
		},
	}

	vod, err := ValuesOrDefaults(vals, b)

	is.NoError(err)
	is.True(vod["enabled"].(bool))
	is.Equal(vod["host"].(string), "localhost")
	is.Equal(vod["port"].(int), 8080)
	is.Equal(vod["replicaCount"].(int), 3)

	// This should err out because of type problem
	vals["replicaCount"] = "banana"
	_, err = ValuesOrDefaults(vals, b)
	is.Error(err)
}

func TestValuesOrDefaults_Required(t *testing.T) {
	is := assert.New(t)
	vals := map[string]interface{}{
		"enabled": true,
	}
	b := &Bundle{
		Definitions: map[string]*definition.Schema{
			"minType": {
				Type: "integer",
			},
			"enabledType": {
				Type:    "boolean",
				Default: false,
			},
		},
		Parameters: ParametersDefinition{
			Fields: map[string]ParameterDefinition{
				"minimum": {
					Definition: "minType",
				},
				"enabled": {
					Definition: "enabledType",
				},
			},
			Required: []string{"minimum"},
		},
	}

	_, err := ValuesOrDefaults(vals, b)
	is.Error(err)

	// It is unclear what the outcome should be when the user supplies
	// empty values on purpose. For now, we will assume those meet the
	// minimum definition of "required", and that other rules will
	// correct for empty values.
	//
	// Example: It makes perfect sense for a user to specify --set minimum=0
	// and in so doing meet the requirement that a value be specified.
	vals["minimum"] = 0
	res, err := ValuesOrDefaults(vals, b)
	is.NoError(err)
	is.Equal(0, res["minimum"])
}

func TestValidateVersionTag(t *testing.T) {
	is := assert.New(t)

	img := InvocationImage{BaseImage{}}
	b := Bundle{
		Version:          "latest",
		InvocationImages: []InvocationImage{img},
	}

	err := b.Validate()
	is.EqualError(err, "'latest' is not a valid bundle version")
}

func TestValidateBundle_RequiresInvocationImage(t *testing.T) {
	b := Bundle{
		Name:    "bar",
		Version: "0.1.0",
	}

	err := b.Validate()
	if err == nil {
		t.Fatal("Validate should have failed because the bundle has no invocation images")
	}

	b.InvocationImages = append(b.InvocationImages, InvocationImage{})

	err = b.Validate()
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadCustomExtensions(t *testing.T) {
	data, err := ioutil.ReadFile("../testdata/bundles/foo.json")
	if err != nil {
		t.Errorf("cannot read bundle file: %v", err)
	}

	bundle, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(bundle.Custom) != 2 {
		t.Errorf("Expected 2 custom extensions, got %d", len(bundle.Custom))
	}

	duffleExtI, ok := bundle.Custom["com.example.duffle-bag"]
	if !ok {
		t.Fatal("Expected the com.example.duffle-bag extension")
	}
	duffleExt, ok := duffleExtI.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected the com.example.duffle-bag to be of type map[string]interface{} but got %T ", duffleExtI)
	}
	assert.Equal(t, "PNG", duffleExt["iconType"])
	assert.Equal(t, "https://example.com/icon.png", duffleExt["icon"])

	backupExtI, ok := bundle.Custom["com.example.backup-preferences"]
	if !ok {
		t.Fatal("Expected the com.example.backup-preferences extension")
	}
	backupExt, ok := backupExtI.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected the com.example.backup-preferences to be of type map[string]interface{} but got %T ", backupExtI)
	}
	assert.Equal(t, true, backupExt["enabled"])
	assert.Equal(t, "daily", backupExt["frequency"])
}

func TestOutputs_Marshall(t *testing.T) {
	bundleJSON := `
	{
		"outputs": {
			"fields" : {
				"clientCert": {
					"contentEncoding": "base64",
					"contentMediaType": "application/x-x509-user-cert",
					"path": "/cnab/app/outputs/clientCert",
					"definition": "clientCert"
				},
				"hostName": {
					"applyTo": [
					"install"
					],
					"description": "the hostname produced installing the bundle",
					"path": "/cnab/app/outputs/hostname",
					"definition": "hostType"
				},
				"port": {
					"path": "/cnab/app/outputs/port",
					"definition": "portType"
				}
			}
		}
	}`

	bundle, err := Unmarshal([]byte(bundleJSON))
	assert.NoError(t, err, "should have unmarshalled the bundle")
	require.NotNil(t, bundle.Outputs, "test must fail, not outputs found")
	assert.Equal(t, 3, len(bundle.Outputs.Fields))

	clientCert, ok := bundle.Outputs.Fields["clientCert"]
	require.True(t, ok, "expected clientCert to exist as an output")
	assert.Equal(t, "clientCert", clientCert.Definition)
	assert.Equal(t, "/cnab/app/outputs/clientCert", clientCert.Path, "clientCert path was not the expected value")

	hostName, ok := bundle.Outputs.Fields["hostName"]
	require.True(t, ok, "expected hostname to exist as an output")
	assert.Equal(t, "hostType", hostName.Definition)
	assert.Equal(t, "/cnab/app/outputs/hostname", hostName.Path, "hostName path was not the expected value")

	port, ok := bundle.Outputs.Fields["port"]
	require.True(t, ok, "expected port to exist as an output")
	assert.Equal(t, "portType", port.Definition)
	assert.Equal(t, "/cnab/app/outputs/port", port.Path, "port path was not the expected value")
}

func TestBundleMarshallAllThings(t *testing.T) {
	cred := Credential{
		Description: "a password",
	}
	cred.EnvironmentVariable = "PASSWORD"
	cred.Path = "/cnab/app/path"

	b := &Bundle{
		SchemaVersion: "v1.0.0-WD",
		Name:          "testBundle",
		Description:   "something",
		Version:       "1.0",
		Credentials: map[string]Credential{
			"password": cred,
		},
		Images: map[string]Image{
			"server": {
				BaseImage: BaseImage{
					Image:     "nginx:1.0",
					ImageType: "docker",
				},
				Description: "complicated",
			},
		},
		InvocationImages: []InvocationImage{
			{
				BaseImage: BaseImage{
					Image:     "deislabs/invocation-image:1.0",
					ImageType: "docker",
				},
			},
		},
		Definitions: map[string]*definition.Schema{
			"portType": {
				Type:    "integer",
				Default: 1234,
			},
			"hostType": {
				Type:    "string",
				Default: "locahost.localdomain",
			},
			"replicaCountType": {
				Type:    "integer",
				Default: 3,
			},
			"enabledType": {
				Type:    "boolean",
				Default: false,
			},
			"clientCert": {
				Type:            "string",
				ContentEncoding: "base64",
			},
		},
		Parameters: ParametersDefinition{
			Fields: map[string]ParameterDefinition{
				"port": {
					Definition: "portType",
					Destination: &Location{
						EnvironmentVariable: "PORT",
					},
				},
				"host": {
					Definition: "hostType",
					Destination: &Location{
						EnvironmentVariable: "HOST",
					},
				},
				"enabled": {
					Definition: "enabledType",
					Destination: &Location{
						EnvironmentVariable: "ENABLED",
					},
				},
				"replicaCount": {
					Definition: "replicaCountType",
					Destination: &Location{
						EnvironmentVariable: "REPLICA_COUNT",
					},
				},
			},
			Required: []string{"port", "host"},
		},
		Outputs: OutputsDefinition{
			Fields: map[string]OutputDefinition{
				"clientCert": {
					Path:       "/cnab/app/outputs/blah",
					Definition: "clientCert",
				},
			},
		},
	}

	expectedJSON, err := ioutil.ReadFile("../testdata/bundles/canonical-bundle.json")
	require.NoError(t, err, "couldn't read test data")

	var buf bytes.Buffer

	_, err = b.WriteTo(&buf)
	require.NoError(t, err, "test requires output")
	assert.Equal(t, []byte(expectedJSON), buf.Bytes(), "output should match expected canonical json")
}

func TestValidateABundleAndParams(t *testing.T) {

	bun, err := ioutil.ReadFile("../testdata/bundles/foo.json")
	require.NoError(t, err, "couldn't read test bundle")

	bundle, err := Unmarshal(bun)
	require.NoError(t, err, "the bundle should have been valid")

	def, ok := bundle.Definitions["complexThing"]
	require.True(t, ok, "test failed because definition not found")

	testData := struct {
		Port int    `json:"port"`
		Host string `json:"hostName"`
	}{
		Host: "validhost",
		Port: 8080,
	}
	valErrors, err := def.Validate(testData)
	assert.NoError(t, err, "validation should not have resulted in an error")
	assert.Empty(t, valErrors, "validation should have been successful")

	testData2 := struct {
		Host string `json:"hostName"`
	}{
		Host: "validhost",
	}
	valErrors, err = def.Validate(testData2)
	assert.NoError(t, err, "validation should not have encountered an error")
	assert.NotEmpty(t, valErrors, "validation should not have been successful")

	testData3 := struct {
		Port int    `json:"port"`
		Host string `json:"hostName"`
	}{
		Host: "validhost",
		Port: 80,
	}
	valErrors, err = def.Validate(testData3)
	assert.NoError(t, err, "should not have encountered an error with the validator")
	assert.NotEmpty(t, valErrors, "validation should not have been successful")
}