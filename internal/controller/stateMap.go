package controller

import (
	"context"
	"fmt"
	k8sCRLib "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/logging"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type StateMap struct {
	ctx                context.Context
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
		ctx:                ctx,
		configMapInterface: configMapInterface,
		debugCR:            debugCR,
		logger:             logging.FromContext(ctx),
	}
	stateMap.configMap = stateMap.getOrCreateConfigMap()
	return stateMap
}

func (s *StateMap) Destroy() (bool, error) {
	cmName := "debugmode-state"
	_, err := s.configMapInterface.Get(s.ctx, cmName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			// generic error - not would be ok
			return false, err
		}
		// statemap does not exists
		return false, nil
	}

	err = s.configMapInterface.Delete(s.ctx, cmName, metav1.DeleteOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *StateMap) getOrCreateConfigMap() *corev1.ConfigMap {
	cmName := "debugmode-state"
	cm, err := s.configMapInterface.Get(s.ctx, cmName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			// generic error - not would be ok
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

		if cm, err = s.configMapInterface.Create(s.ctx, cm, metav1.CreateOptions{}); err != nil {
			s.logger.Error(fmt.Sprintf("ERROR: failed to create configMap: %v", err))
			return nil
		}
	}

	return cm
}

func (s *StateMap) compareWithStateMap(key string, target string) (current string, equals bool) {
	val, ok := s.configMap.Data[key]
	if !ok {
		return "", false
	}
	return val, strings.EqualFold(val, target)
}

func (s *StateMap) updateStateMap(key string, value string) error {
	s.logger.Debug(fmt.Sprintf("Update state map %s:%s", key, value))
	if s.configMap.Data == nil {
		s.logger.Debug(fmt.Sprintf("- create new configmap data"))
		s.configMap.Data = map[string]string{}
	}

	s.configMap.Data[key] = value

	s.logger.Debug(fmt.Sprintf("- start update"))
	newMap, err := s.configMapInterface.Update(s.ctx, s.configMap, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to Update configMap: %v", err))
	}
	s.logger.Debug(fmt.Sprintf("- updated map %s:%s to %v", key, value, newMap))
	s.configMap = newMap
	return err
}
