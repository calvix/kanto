// Kanto
// web service to manage and scale couchdb running on kubernetes
// author: Vaclav Rozsypalek
// Created on 21.06.2016

// mostly dummy functions for handling operation with users
// ie authentication, validation etc
package kanto

import "net/http"

type User struct {
	UserName string `json:"username"`
	Token    string `json:"token"`
	// TODO
}

// valid username and its token against database
// DUMMY function, not implemented
// @param none
// @return bool - true if authentication was successful
func (u *User) IsAuthenticated() bool {
	// DUMMY function, not implemented
	// DUMMY
	return true
}

// parse user from HTTP POST request
// and return initialized User struct
// @param r * http.Request - request send to REST API with credentials included
// @return u - fully initialised user Struct with data from http request
func ParseUser(r *http.Request) (u *User) {
	// parse data from POST request
	username := r.FormValue("username")
	token := r.FormValue("token")

	// init a user struct
	u = &User{UserName: username, Token: token}

	// done
	return u
}
// dummy db storage
var dbs_storage map[string][]string = make(map[string][]string)
func DatabasesToReplicate(username string) ([]string){
	// DUMMY
	users_db := dbs_storage[username]
	if len(users_db) > 0 {
		return users_db
	} else {
		return []string{"test", "_users"}
	}


}

func SaveReplDatabases(username string, dbs []string) {
	// DUMMY
	users_db := dbs_storage[username]
	if len(users_db) > 0 {
		dbs_storage[username] = append(users_db, dbs...)
	} else {
		dbs_storage[username] = append([]string{"test", "_users"}, dbs...)
	}
}
