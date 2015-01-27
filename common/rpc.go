package common

type DiffRequest struct {
	ServerPath string
	MD5 bool
	Index Index
}

type DiffResponse struct {
	SessionID string
	Upd []string
}

type UpdateResponse struct {
	Done bool
}
