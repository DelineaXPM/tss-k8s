package injector

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/mattbaird/jsonpatch"
	"github.com/thycotic/tss-sdk-go/server"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
roleAnnotation contains a role that maps to a set of credentials in Roles (see below)
setAnnotation, addAnnoation and updateAnnotation contain the path to the TSS Secret
that will be used to modified this Secret.
addAnnotation adds missing fields without overrwriting or removing existing fields
updateAnnotation adds and overwrites existing fields but does not remove fields
setAnnotation overwrites fields and removes fields that do not exist in the TSS Secret
*/
const (
	roleAnnotation   = "tss.thycotic.com/role"
	setAnnotation    = "tss.thycotic.com/set-secret"
	addNotation      = "tss.thycotic.com/add-to-secret"
	updateAnnotation = "tss.thycotic.com/update-secret"
	tsAnnotation     = "tss.thycotic.com/modified"
)

/*
noPatch causes Inject to approve without patching
patchAdd causes Inject to add entries without overwriting or removing existing ones
patchUpdate causes Inject to add and update entries without removing existing ones
patchOverwrite causes Inject to completely overwrite all entries including removal
of existing entries that are not present in the TSS Secret
*/
const (
	noPatch = iota
	patchAdd
	patchUpdate
	patchOverwrite
)

// Roles is a mapping of roleName to tss-sdk-go/server/Configuration objects
type Roles map[string]struct {
	server.Configuration
}

// Inject adds to, updates or replaces the k8s Secret.Data with tss Secret.Fields (see above)
func Inject(ar *v1beta1.AdmissionReview, roles Roles) error {
	ar.Response = &v1beta1.AdmissionResponse{
		Allowed: true,
		Result: &metav1.Status{
			Status: metav1.StatusSuccess,
		},
		UID: ar.Request.UID,
	}

	var config server.Configuration
	var secret corev1.Secret

	if err := json.Unmarshal(ar.Request.Object.Raw, &secret); err != nil {
		return fmt.Errorf("unable to unmarshal Secret: %s", err)
	}
	log.Printf("[DEBUG] operating on k8s Secret '%s'", secret.Name)

	annotations := secret.ObjectMeta.GetAnnotations()
	/*
		If there is a role annotation, use the configuration that corresponds
		to it and return an error if there's no configuration for that role.
		Otherwise use the default role and return an error if there is no
		configuration corresponding to it.
	*/
	if roleName, ok := annotations[roleAnnotation]; ok {
		if role, ok := roles[roleName]; ok {
			config = role.Configuration
		} else {
			return fmt.Errorf("no configuration for role: %s", roleName)
		}
	} else if role, ok := roles["default"]; ok {
		config = role.Configuration
	} else {
		return fmt.Errorf("no %s and no default", roleAnnotation)
	}

	patchMode := noPatch
	var aSecretID string
	var ok bool

	if aSecretID, ok = annotations[setAnnotation]; ok {
		patchMode = patchOverwrite
	} else if aSecretID, ok = annotations[addNotation]; ok {
		patchMode = patchAdd
	} else if aSecretID, ok = annotations[updateAnnotation]; ok {
		patchMode = patchUpdate
	}

	if patchMode != noPatch {
		jsonPatch := []jsonpatch.JsonPatchOperation{
			{
				Operation: "add",
				Path:      "/metadata/annotations",
				Value: map[string]string{
					tsAnnotation: time.Now().UTC().Format(time.UnixDate),
				},
			},
		}

		server, err := server.New(config)

		if err != nil {
			return fmt.Errorf("configuration error: %s", err)
		}

		var secretID int

		if secretID, err = strconv.Atoi(aSecretID); err != nil {
			return fmt.Errorf("secretID must be an integer")
		}

		serverSecret, err := server.Secret(secretID)

		if err != nil {
			return fmt.Errorf("unable to get the secret with ID: %d: %s", secretID, err)
		}

		serverSecretData := map[string]interface{}{}

		for _, field := range serverSecret.Fields {
			serverSecretData[field.Slug] = field.ItemValue
		}
		/*
			CreatePatch returns the difference between the JSON represenation of
			the TSS Secret Data and the k8s Secret Data, as an RFC 6902 JSON Patch.
		*/
		ssdj, _ := json.Marshal(serverSecretData)
		sdj, _ := json.Marshal(secret.Data)
		diff, _ := jsonpatch.CreatePatch(sdj, ssdj)
		/*
			Each patch operation has to be updated so that k8s can apply it to
			the entire Secret:
			1) the Path must be relative to /Secret rather than /Secret/Data
			2) the Values must be Base64 encoded
			3) the Operations that conflict with patchMode must be removed
		*/
		for _, op := range diff {
			op.Path = "/data" + op.Path

			switch v := op.Value.(type) {
			case []byte:
				op.Value = base64.StdEncoding.EncodeToString(v)
			case string:
				op.Value = base64.StdEncoding.EncodeToString([]byte(v))
			}

			switch op.Operation {
			case "replace":
				if patchMode == patchAdd {
					continue
				}
			case "remove":
				if patchMode == patchAdd || patchMode == patchUpdate {
					continue
				}
			}
			jsonPatch = append(jsonPatch, op)
		}

		patchType := v1beta1.PatchTypeJSONPatch
		patch, err := json.Marshal(jsonPatch)

		if err != nil {
			return fmt.Errorf("unable to marshal JsonPatch: %s", err)
		}

		for i := range jsonPatch {
			jsonPatch[i].Value = "*omitted*" // omit values in the DEBUG log
		}
		log.Printf("[DEBUG] patching the Secret with %s", jsonPatch)

		ar.Response.PatchType = &patchType
		ar.Response.Patch = patch
	}
	return nil
}
