package controller

import (
	"context"
	"fmt"
	k8sCRLib "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"strings"
)

type StateMap struct {
	ctx                context.Context
	debugCR            *k8sCRLib.DebugMode
	configMapInterface typev1.ConfigMapInterface
	logger             logr.Logger
	configMap          *corev1.ConfigMap
}

func NewStateMap(ctx context.Context,
	debugCR *k8sCRLib.DebugMode,
	configMapInterface typev1.ConfigMapInterface,
	logger logr.Logger) *StateMap {
	stateMap := &StateMap{
		ctx:                ctx,
		configMapInterface: configMapInterface,
		logger:             logger,
		debugCR:            debugCR,
	}
	stateMap.configMap = stateMap.getOrCreateConfigMap()
	return stateMap
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
			s.logger.Info(fmt.Sprintf("ERROR: failed to create configMap: %w", err))
			return nil
		}
	}

	return cm
}

func (s *StateMap) compareWithStateMap(key string, target string) (current string, equals bool) {
	s.logger.Info(fmt.Sprintf("Compare with state map %s:%s", key, target))
	defer func() {
		s.logger.Info(fmt.Sprintf(" - current: %s, equals: %b", current, equals))
	}()
	val, ok := s.configMap.Data[key]
	if !ok {
		return "", false
	}
	return val, strings.EqualFold(val, target)
}

func (s *StateMap) updateStateMap(key string, value string) error {
	s.logger.Info(fmt.Sprintf("Update state map %s:%s", key, value))
	if s.configMap.Data == nil {
		s.logger.Info(fmt.Sprintf("- create new configmap data"))
		s.configMap.Data = map[string]string{}
	}

	s.configMap.Data[key] = value

	s.logger.Info(fmt.Sprintf("- start update"))
	newMap, err := s.configMapInterface.Update(s.ctx, s.configMap, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Info(fmt.Sprintf("Failed to Update configMap: %w", err))
	}
	s.logger.Info(fmt.Sprintf("- updated map %s:%s to %v", key, value, newMap))
	s.configMap = newMap
	return err
}
