package habgrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/naming"
)

type resolver struct {
}

type watcher struct {
	serviceGroup string
	portCfgKey   string
	lastTargets  []string
}

func NewResolver() naming.Resolver {
	return &resolver{}
}

func (r *resolver) Resolve(target string) (naming.Watcher, error) {
	parts := strings.Split(target, ":")
	if len(parts) == 2 {
		return &watcher{parts[0], parts[1], []string{}}, nil
	} else {
		return nil, errors.New("Could not parse target")
	}
}

func (w *watcher) Close() {
}

func (w *watcher) Next() ([]*naming.Update, error) {
	for {
		newTargets, err := getTargets(w.serviceGroup, w.portCfgKey)

		if err == nil {
			operations := []*naming.Update{}

			for _, targetNew := range newTargets {
				found := false
				for _, targetOld := range w.lastTargets {
					if targetOld == targetNew {
						found = true
						break
					}
				}

				if !found {
					operations = append(operations, &naming.Update{
						Addr: targetNew,
						Op:   naming.Add,
					})
				}
			}

			for _, targetOld := range w.lastTargets {
				found := false
				for _, targetNew := range newTargets {
					if targetOld == targetNew {
						found = true
						break
					}
				}

				if !found {
					operations = append(operations, &naming.Update{
						Addr: targetOld,
						Op:   naming.Delete,
					})
				}
			}
			if len(operations) > 0 {
				w.lastTargets = newTargets
				return operations, nil
			}
		} else {
			logrus.WithError(err).Error("Failed to get census data")
		}

		time.Sleep(10 * time.Second)
	}
}

type CensusResponse struct {
	CensusGroups map[string]CensusGroup `json:"census_groups"`
}

type CensusGroup struct {
	Population map[string]MemberInfo `json:"population"`
}

type MemberInfo struct {
	MemberId string                 `json:"member_id"`
	Sys      MemberSys              `json:"sys"`
	Cfg      map[string]interface{} `json:"cfg"`
}

type MemberSys struct {
	Ip string `json:"ip"`
}

func getTargets(serviceGroup string, portCfgKey string) ([]string, error) {
	resp, err := http.Get("http://localhost:9631/census")
	if err != nil {
		return nil, err
	}

	var censusResp CensusResponse

	if err := json.NewDecoder(resp.Body).Decode(&censusResp); err != nil {
		return nil, err
	}

	targets := []string{}

	if group, ok := censusResp.CensusGroups[serviceGroup]; ok {
		for _, memberInfo := range group.Population {
			if port, ok := memberInfo.Cfg[portCfgKey]; ok {
				targets = append(targets, fmt.Sprintf("%v:%v", memberInfo.Sys.Ip, port))
			}
		}
	}

	return targets, nil
}
