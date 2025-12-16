package controller

import (
	"context"
	"fmt"

	k8sCRLib "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/logging"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DEFAULT_CM_NAME = "debugmode-state"

type StateMap struct {
	debugCR            *k8sCRLib.DebugMode
	configMapInterface configurationMap
	logger             logging.Logger
	configMap          *corev1.ConfigMap
}

func NewStateMap(ctx context.Context,
	debugCR *k8sCRLib.DebugMode,
	configMapInterface configurationMap,
) *StateMap {
	stateMap := &StateMap{
		configMapInterface: configMapInterface,
		debugCR:            debugCR,
		logger:             logging.FromContext(ctx),
	}
	stateMap.configMap = stateMap.getOrCreateConfigMap(ctx)
	return stateMap
}

func (s *StateMap) Destroy(ctx context.Context) (bool, error) {
	cmName := DEFAULT_CM_NAME
	_, err := s.configMapInterface.Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			// generic error - not would be ok
			return false, err
		}
		// statemap does not exists
		return false, nil
	}

	err = s.configMapInterface.Delete(ctx, cmName, metav1.DeleteOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *StateMap) getOrCreateConfigMap(ctx context.Context) *corev1.ConfigMap {
	cmName := DEFAULT_CM_NAME
	cm, err := s.configMapInterface.Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			// generic error - not would be ok
			return nil
		}
		if s.debugCR == nil {
			// do not create state map when CR is deleted - this should lead to an error
			return nil
		}
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: s.debugCR.Namespace,
				Labels: map[string]string{
					"debugmode.k8s.cloudogu.com/owner": s.debugCR.Name,
				},
			},
			Data: map[string]string{},
		}

		if cm, err = s.configMapInterface.Create(ctx, cm, metav1.CreateOptions{}); err != nil {
			s.logger.Error(fmt.Sprintf("ERROR: failed to create configMap: %v", err))
			return nil
		}
	}

	return cm
}

func (s *StateMap) getValueFromMap(key string) string {
	val, ok := s.configMap.Data[key]
	if !ok {
		return ""
	}
	return val
}

func (s *StateMap) updateStateMap(ctx context.Context, key string, value string) error {
	s.logger.Debug(fmt.Sprintf("Update state map %s:%s", key, value))
	if s.configMap.Data == nil {
		s.logger.Debug("- create new configmap data")
		s.configMap.Data = map[string]string{}
	}

	s.configMap.Data[key] = value

	s.logger.Debug("- start update")
	newMap, err := s.configMapInterface.Update(ctx, s.configMap, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to Update configMap: %v", err))
	}
	s.logger.Debug(fmt.Sprintf("- updated map %s:%s to %v", key, value, newMap))
	s.configMap = newMap
	return err
}
