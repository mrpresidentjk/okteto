package model

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/okteto/okteto/pkg/log"
	yaml "gopkg.in/yaml.v2"
	apiv1 "k8s.io/api/core/v1"
)

// Dev represents a development container
type DevRC struct {
	Autocreate           *bool                 `json:"autocreate,omitempty" yaml:"autocreate,omitempty"`
	Labels               Labels                `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations          Annotations           `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Context              string                `json:"context,omitempty" yaml:"context,omitempty"`
	Namespace            string                `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	ImagePullPolicy      apiv1.PullPolicy      `json:"imagePullPolicy,omitempty" yaml:"imagePullPolicy,omitempty"`
	Environment          Environment           `json:"environment,omitempty" yaml:"environment,omitempty"`
	Secrets              []Secret              `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	Healthchecks         *bool                 `json:"healthchecks,omitempty" yaml:"healthchecks,omitempty"`
	SecurityContext      *SecurityContext      `json:"securityContext,omitempty" yaml:"securityContext,omitempty"`
	ServiceAccount       string                `json:"serviceAccount,omitempty" yaml:"serviceAccount,omitempty"`
	Sync                 Sync                  `json:"sync,omitempty" yaml:"sync,omitempty"`
	Interface            string                `json:"interface,omitempty" yaml:"interface,omitempty"`
	Resources            ResourceRequirements  `json:"resources,omitempty" yaml:"resources,omitempty"`
	PersistentVolumeInfo *PersistentVolumeInfo `json:"persistentVolume,omitempty" yaml:"persistentVolume,omitempty"`
	InitFromImage        *bool                 `json:"initFromImage,omitempty" yaml:"initFromImage,omitempty"`
	Timeout              time.Duration         `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// Get returns a Dev object from a given file
func GetRc(devPath string) (*DevRC, error) {
	b, err := ioutil.ReadFile(devPath)
	if err != nil {
		return nil, err
	}

	dev, err := ReadRC(b)
	if err != nil {
		log.Warning("ignoring developer overwrites defined in %s: %s", devPath, err.Error())
		return nil, err
	}

	return dev, nil
}

// Read reads an okteto manifests
func ReadRC(bytes []byte) (*DevRC, error) {
	dev := &DevRC{}

	if bytes != nil {
		if err := yaml.UnmarshalStrict(bytes, dev); err != nil {
			if strings.HasPrefix(err.Error(), "yaml: unmarshal errors:") {
				var sb strings.Builder
				_, _ = sb.WriteString("Invalid manifest:\n")
				l := strings.Split(err.Error(), "\n")
				for i := 1; i < len(l); i++ {
					e := strings.TrimSuffix(l[i], "in type model.Dev")
					e = strings.TrimSpace(e)
					_, _ = sb.WriteString(fmt.Sprintf("    - %s\n", e))
				}

				_, _ = sb.WriteString("    See https://okteto.com/docs/reference/manifest for details")
				return nil, errors.New(sb.String())
			}

			msg := strings.Replace(err.Error(), "yaml: unmarshal errors:", "invalid manifest:", 1)
			msg = strings.TrimSuffix(msg, "in type model.Dev")
			return nil, errors.New(msg)
		}
	}

	return dev, nil
}

func MergeDevWithDevRc(dev *Dev, devRc *DevRC) {
	if devRc.Autocreate != nil && !dev.Autocreate {
		dev.Autocreate = *devRc.Autocreate
	}

	for labelKey, labelValue := range devRc.Labels {
		if _, ok := dev.Labels[labelKey]; !ok {
			dev.Labels[labelKey] = labelValue
		}
	}

	for annotationKey, annotationValue := range devRc.Annotations {
		if _, ok := dev.Annotations[annotationKey]; !ok {
			dev.Annotations[annotationKey] = annotationValue
		}
	}

	if devRc.Context != "" && dev.Context == "" {
		dev.Context = devRc.Context
	}

	if devRc.Namespace != "" && dev.Namespace == "" {
		dev.Namespace = devRc.Namespace
	}

	if devRc.ImagePullPolicy != "" && dev.ImagePullPolicy == "" {
		dev.ImagePullPolicy = devRc.ImagePullPolicy
	}

	for _, env := range devRc.Environment {
		if !isEnvOnDev(dev, env) {
			dev.Environment = append(dev.Environment, env)
		}
	}

	for _, secret := range devRc.Secrets {
		if !isSecretOnDev(dev, secret) {
			dev.Secrets = append(dev.Secrets, secret)
		}
	}

	if devRc.Healthchecks != nil && !dev.Healthchecks {
		dev.Healthchecks = *devRc.Healthchecks
	}

	if devRc.SecurityContext != nil && dev.SecurityContext == nil {
		dev.SecurityContext = devRc.SecurityContext
	} else if devRc.SecurityContext != nil && dev.SecurityContext != nil {
		if devRc.SecurityContext.FSGroup != nil && dev.SecurityContext.FSGroup == nil {
			dev.SecurityContext.FSGroup = devRc.SecurityContext.FSGroup
		}
		if devRc.SecurityContext.RunAsGroup != nil && dev.SecurityContext.RunAsGroup == nil {
			dev.SecurityContext.RunAsGroup = devRc.SecurityContext.RunAsGroup
		}
		if devRc.SecurityContext.RunAsUser != nil && dev.SecurityContext.RunAsUser == nil {
			dev.SecurityContext.RunAsUser = devRc.SecurityContext.RunAsUser
		}
		for _, cap := range devRc.SecurityContext.Capabilities.Add {
			if !isCapabilityInDev(dev.SecurityContext.Capabilities.Add, cap) {
				dev.SecurityContext.Capabilities.Add = append(dev.SecurityContext.Capabilities.Add, cap)
			}
		}

		for _, cap := range devRc.SecurityContext.Capabilities.Drop {
			if !isCapabilityInDev(dev.SecurityContext.Capabilities.Drop, cap) {
				dev.SecurityContext.Capabilities.Drop = append(dev.SecurityContext.Capabilities.Drop, cap)
			}
		}
	}

	if devRc.ServiceAccount != "" && dev.ServiceAccount == "" {
		dev.ServiceAccount = devRc.ServiceAccount
	}

	if devRc.Sync.Compression && !dev.Sync.Compression {
		dev.Sync.Compression = devRc.Sync.Compression
	}

	if devRc.Sync.Verbose && !dev.Sync.Verbose {
		dev.Sync.Verbose = devRc.Sync.Verbose
	}

	if devRc.Sync.RescanInterval != 0 && dev.Sync.RescanInterval == 0 {
		dev.Sync.RescanInterval = devRc.Sync.RescanInterval
	}

	for _, folder := range devRc.Sync.Folders {
		if !isFolderSyncInDev(dev.Sync.Folders, folder) {
			dev.Sync.Folders = append(dev.Sync.Folders, folder)
		}
	}

	if devRc.Interface != "" && dev.Interface == "" {
		dev.Interface = devRc.Interface
	}

	for resourceKey, resourceValue := range devRc.Resources.Limits {
		if _, ok := dev.Resources.Limits[resourceKey]; !ok {
			dev.Resources.Limits[resourceKey] = resourceValue
		}
	}

	for resourceKey, resourceValue := range devRc.Resources.Requests {
		if _, ok := dev.Resources.Requests[resourceKey]; !ok {
			dev.Resources.Requests[resourceKey] = resourceValue
		}
	}

	if devRc.PersistentVolumeInfo != nil && dev.PersistentVolumeInfo == nil {
		dev.PersistentVolumeInfo = devRc.PersistentVolumeInfo
	} else if devRc.PersistentVolumeInfo != nil && dev.PersistentVolumeInfo != nil {
		if devRc.PersistentVolumeInfo.Size != "" && dev.PersistentVolumeInfo.Size == "" {
			dev.PersistentVolumeInfo.Size = devRc.PersistentVolumeInfo.Size
		}
		if devRc.PersistentVolumeInfo.StorageClass != "" && dev.PersistentVolumeInfo.StorageClass == "" {
			dev.PersistentVolumeInfo.StorageClass = devRc.PersistentVolumeInfo.StorageClass
		}
	}

	if devRc.InitFromImage != nil && !dev.InitFromImage {
		dev.InitFromImage = *devRc.InitFromImage
	}
	if devRc.InitFromImage != nil && !dev.InitFromImage {
		dev.InitFromImage = *devRc.InitFromImage
	}

	if devRc.Timeout != 0 && dev.Timeout == 0 {
		dev.Timeout = devRc.Timeout
	}
}

func isEnvOnDev(dev *Dev, env EnvVar) bool {
	for _, devEnv := range dev.Environment {
		if devEnv.Name == env.Name {
			return true
		}
	}
	return false
}

func isSecretOnDev(dev *Dev, secret Secret) bool {
	for _, devSecret := range dev.Secrets {
		if devSecret.LocalPath == secret.LocalPath || devSecret.RemotePath == secret.RemotePath {
			return true
		}
	}
	return false
}

func isCapabilityInDev(devCapabilities []apiv1.Capability, cap apiv1.Capability) bool {
	for _, devCap := range devCapabilities {
		if devCap == cap {
			return true
		}
	}
	return false
}

func isFolderSyncInDev(devSyncFolder []SyncFolder, folder SyncFolder) bool {
	for _, devSyncFolder := range devSyncFolder {
		if devSyncFolder.LocalPath == folder.LocalPath && devSyncFolder.RemotePath == folder.RemotePath {
			return true
		}
	}
	return false
}