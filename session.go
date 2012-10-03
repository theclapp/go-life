package main

// vim:sw=3:ts=3:fdm=indent

import (
	"fmt"
	"net/http"
	"sync"
)

type sessionId string

type session struct {
	sid        sessionId
	nextPageId int
	// map pageIds to listener channels
	listeners map[string]chan string
	delay     float32
	stop      int
	uCh       chan *universe
	u         *universe
	numU      int
	gen       int
	sync.Mutex
}

var sessions = make(map[sessionId]*session)
var sessionsLock sync.Mutex
var nextSessionNum = 0

// FIXME Locks sessions[]: add a new session.
func getSession(w http.ResponseWriter, req *http.Request) *session {
	var sid sessionId
	sessionsLock.Lock()
	defer sessionsLock.Unlock()
	sessionCookie, err := req.Cookie("session")
	if err == nil {
		sid = sessionId(sessionCookie.Value) // type conversion
		if sessions[sid] == nil {
			fmt.Printf("%s: Invalid session cookie: %s\n", curFuncName(1), sid)
		} else {
			return sessions[sid]
		}
	}

	fmt.Printf("%s: No session cookie; setting %d\n", curFuncName(1), nextSessionNum)
	sid = sessionId(fmt.Sprintf("%d", nextSessionNum)) // type conversion
	sessionCookie = &http.Cookie{
		Name:   "session",
		Value:  string(sid),
		MaxAge: 3600,
	}
	http.SetCookie(w, sessionCookie)
	sessions[sid] = &session{
		sid:       sid,
		listeners: make(map[string]chan string),
		delay:     800.00,
		uCh:       make(chan *universe),
	}
	nextSessionNum++
	go func(s *session) {
		u := makeUniverse()
		for {
			s.uCh <- u
			u = nextGen(u)
		}
	}(sessions[sid])
	return sessions[sid]
}

func curFuncName(n int) string {
	pc, _, _, _ := runtime.Caller(n + 1)
	return runtime.FuncForPC(pc).Name()
}

// FIXME Locks the session: get the next PageId
func (s *session) getNextPageId() (result string) {
	s.Lock()
	defer s.Unlock()
	result = fmt.Sprintf("%d", s.nextPageId)
	s.nextPageId++
	return
}

// FIXME Locks the session: Get the next Universe.
func (s *session) nextU(pageId string) *universe {
	s.Lock()
	defer s.Unlock()
	if s.numU < len(s.listeners) {
		s.numU++
	} else {
		s.u = <-s.uCh
		s.gen++
		s.numU = 1
	}
	fmt.Printf("session: %s, pageId: %s, gen: %d, numU: %d, #listeners: %d\n",
		s.sid, pageId, s.gen, s.numU, len(s.listeners))
	return s.u
}

// FIXME Locks the session, and sometimes sessions[]: change delay and/or
// stop, or delete the session entirely.
func (s *session) pressButton(w http.ResponseWriter, whichButton string) {
	s.Lock()
	defer s.Unlock()
	fmt.Printf("title is %s\n", whichButton)
	switch whichButton {
	case "delayMore":
		s.delay *= 2
	case "delayLess":
		s.delay /= 2
	case "stopLife":
		s.stop = 1
		s.delay++
	case "startLife":
		s.stop = 0
		s.delay--
	case "clearSession":
		sessionsLock.Lock()
		delete(sessions, s.sid)
		sessionsLock.Unlock()
		http.SetCookie(w, &http.Cookie{
			Name:   "session",
			MaxAge: -1,
		})
	}
}

// FIXME Locks the session: add a new listener
func (s *session) addListener(pageId string) chan string {
	s.Lock()
	defer s.Unlock()
	s.listeners[pageId] = make(chan string)
	return s.listeners[pageId]
}

// FIXME Locks the session: delete the listener when the event's sent.
func (s *session) removeListener(pageId string) {
	s.Lock()
	defer s.Unlock()
	delete(s.listeners, pageId)
}
