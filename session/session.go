package session

import (
	"errors"
	"github.com/google/uuid"
	"time"
)

type Session struct {
	Id            string
	UserId        string
	Role          string
	NotValidAfter int64
	AuthBackend   string
}

type Store interface {
	Get(string) (Session, error)
	Store(Session)
	Delete(string)
	Purge()
}

type MemoryStore struct {
	sessions map[string]Session
}

func CreateSession(userId, role, authBackend string, lifetime int) Session {
	return Session{
		Id:            uuid.New().String(),
		UserId:        userId,
		Role:          role,
		NotValidAfter: time.Now().Unix() + int64(lifetime),
		AuthBackend:   authBackend,
	}
}

func IsValid(s *Session) bool {
	return time.Now().Unix() < s.NotValidAfter
}

func NewMemoryStore() Store {
	return &MemoryStore{sessions: make(map[string]Session)}
}

func (s *MemoryStore) Get(id string) (Session, error) {
	val, ok := s.sessions[id]
	if ok {
		return val, nil
	}
	return Session{}, errors.New("session not found")
}

func (s *MemoryStore) Store(session Session) {
	s.sessions[session.Id] = session
}

func (s *MemoryStore) Delete(id string) {
	delete(s.sessions, id)
}

func (s *MemoryStore) Purge() {
	for id, sess := range s.sessions {
		if !IsValid(&sess) {
			s.Delete(id)
		}
	}
}
