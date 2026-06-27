package main

import "fmt"

type PolicyEngine struct {
	rules         map[string]string
	defaultAction string
}

func NewPolicyEngine(cfg *Config) *PolicyEngine {
	rules := make(map[string]string, len(cfg.Policies))
	for _, p := range cfg.Policies {
		key := fmt.Sprintf("%s->%s", p.Source, p.Target)
		rules[key] = p.Action
	}
	return &PolicyEngine{rules: rules, defaultAction: cfg.DefaultAction}
}

// Check returns true if the source is allowed to call the target.
func (e *PolicyEngine) Check(source, target string) bool {
	key := fmt.Sprintf("%s->%s", source, target)
	action, ok := e.rules[key]
	if !ok {
		action = e.defaultAction
	}
	return action == "allow"
}

// Rules returns all rules as a map for inspection.
func (e *PolicyEngine) Rules() map[string]string {
	out := make(map[string]string, len(e.rules))
	for k, v := range e.rules {
		out[k] = v
	}
	return out
}
