package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/cilium/ebpf"
)

type PolicyAction string

const (
	Pass   PolicyAction = "pass"
	Block  PolicyAction = "block"
	Ignore PolicyAction = "ignore"
)

func (a PolicyAction) toEBPF() uint64 {
	switch a {
	case Pass:
		return 0
	case Block:
		return 1
	case Ignore:
		return 2
	default:
		return 0
	}
}

func (a PolicyAction) IsValid() bool {
	return a == Pass || a == Block || a == Ignore
}

func fromEBPF(value uint64) PolicyAction {
	switch value {
	case 0:
		return Pass
	case 1:
		return Block
	case 2:
		return Ignore
	default:
		return Pass
	}
}

type Policy struct {
	IP     string       `json:"ip"`
	Action PolicyAction `json:"action"`
}

func (p *Policy) toEBPF() (uint32, []uint64, error) {
	ip := net.ParseIP(p.IP)
	if ip == nil {
		return 0, nil, fmt.Errorf("invalid IP address: %s", p.IP)
	}

	key := ipToUint32(ip)
	singleValue := p.Action.toEBPF()
	value := make([]uint64, cpuCount)
	for i := range value {
		value[i] = singleValue
	}

	return key, value, nil
}

func getPolicyHandler(IpPolicies *ebpf.Map) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		policies := listPolicies(IpPolicies)
		json.NewEncoder(w).Encode(policies)
	}
}

func deletePolicyHandler(IpPolicies *ebpf.Map) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.PathValue("ip")
		if err := removePolicy(IpPolicies, ip); err != nil {
			if errors.Is(err, ebpf.ErrKeyNotExist) {
				log.Warn().Err(err).Str("ip", ip).Msg("Policy does not exist")
				http.Error(w, "Policy not found", http.StatusNotFound)
			} else {
				log.Error().Err(err).Str("ip", ip).Msg("Failed to remove policy")
				http.Error(w, "Bad request", http.StatusBadRequest)
			}
			return
		}
		/* save to disk */
		if err := storeOnDisk(IpPolicies, policiesFile); err != nil {
			log.Error().Err(err).Msg("Failed to store policies on disk")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		log.Info().Str("ip", ip).Msg("Policy removed successfully")
		w.WriteHeader(http.StatusNoContent)
	}
}

func postPolicyHandler(IpPolicies *ebpf.Map) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policy := Policy{}
		if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
			log.Error().Err(err).Msg("Failed to decode request body")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		key, slice, err := policy.toEBPF()
		if err != nil {
			log.Error().Err(err).Msg("Failed to convert policy to eBPF format")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if err := IpPolicies.Update(key, slice, ebpf.UpdateAny); err != nil {
			log.Error().
				Err(err).
				Str("ip", policy.IP).
				Str("action", string(policy.Action)).
				Msg("Failed to update policy")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		/* save to disk */
		if err := storeOnDisk(IpPolicies, policiesFile); err != nil {
			log.Error().Err(err).Msg("Failed to store policies on disk")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Info().
			Str("ip", policy.IP).
			Str("action", string(policy.Action)).
			Msg("Policy updated successfully")
		w.WriteHeader(http.StatusCreated)
	}
}

func listPolicies(IpPolicies *ebpf.Map) []Policy {
	out := make([]Policy, 0)
	it := IpPolicies.Iterate()
	var key uint32
	c := make([]uint64, 0, cpuCount)
	for it.Next(&key, &c) {
		policy := Policy{
			IP:     uint32ToIP(key).String(),
			Action: fromEBPF(c[0]),
		}
		out = append(out, policy)
	}
	return out
}

func removePolicy(IpPolicies *ebpf.Map, ip string) error {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	key := ipToUint32(parsedIP)
	return IpPolicies.Delete(key)
}

func storeOnDisk(IpPolicies *ebpf.Map, path string) error {
	p := listPolicies(IpPolicies)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	if err := encoder.Encode(p); err != nil {
		return err
	}
	return nil
}

func loadFromDisk(IpPolicies *ebpf.Map, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Warn().Str("file", path).Msg("Policies file does not exist, starting with empty policies")
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var policies []Policy
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&policies); err != nil {
		return fmt.Errorf("failed to decode json: %w", err)
	}

	for _, policy := range policies {
		key, slice, err := policy.toEBPF()
		if err != nil {
			return err
		}

		if err := IpPolicies.Update(key, slice, ebpf.UpdateAny); err != nil {
			return err
		}
	}
	return nil
}
