package models

type DetectMsg struct {
	Name  string `json:"name"`
	Entry string `json:"entry"`
	CN    string `json:"cn,omitempty"`
	Hash  string `json:"hash,omitempty"`
	Raw   string `json:"raw,omitempty"`
}
