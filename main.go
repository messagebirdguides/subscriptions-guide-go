package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	messagebird "github.com/messagebird/go-rest-api"
	"github.com/messagebird/go-rest-api/sms"
)

type appContext struct {
	db *map[string]string
}

func main() {
	db := make(map[string]string)
	app := &appContext{
		&db,
	}
	err := app.initDB("subscriberlist.csv")
	if err != nil {
		log.Println(err)
	}

	http.Handle("/", app.landingHandler())
	http.Handle("/webhook", app.updateSubscriptionHandler())

	port := ":8080"
	log.Println("Serving on " + port)
	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalln(err)
	}
}

func (app *appContext) initDB(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return err
	}
	for i := range rows {
		(*app.db)[rows[i][0]] = rows[i][1]
	}
	return nil
}

func (app *appContext) getSubscriberList() ([]string, error) {
	subscribers := []string{}
	for k := range *app.db {
		if (*app.db)[k] == "yes" {
			subscribers = append(subscribers, k)
		} else {
			// log.Println("omitted: " + k)
		}
	}
	return subscribers, nil
}

func (app *appContext) upsertDB(key string, value string) error {
	(*app.db)[key] = value
	return nil
}

func (app *appContext) sendSMS(msgBody string, originator string, recipients []string) error {
	client := messagebird.New("<enter-your-api-key>")
	msg, err := sms.Create(client, originator, recipients, msgBody, nil)
	if err != nil {
		var mbErrors []string
		switch errResp := err.(type) {
		case messagebird.ErrorResponse:
			for _, mbError := range errResp.Errors {
				mbErrors = append(mbErrors, fmt.Sprint(mbError))
			}
		}
		return errors.New(fmt.Sprint(mbErrors[:]))
	}
	log.Println(msg)
	return nil
}

func (app *appContext) landingHandler() http.HandlerFunc {
	templateData := struct {
		SubscriberCount int
		Message         string
	}{}

	subscriberList, err := app.getSubscriberList()
	if err != nil {
		templateData.Message = fmt.Sprintf("Couldn't get subscriber list: %v", err)
	} else {
		templateData.SubscriberCount = len(subscriberList)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			msgBody := r.FormValue("message")
			originator := "MBSender"
			err := app.sendSMS(msgBody, originator, subscriberList)
			if err != nil {
				templateData.Message = fmt.Sprintf("Could not send message: %v", err)
			} else {
				templateData.Message = fmt.Sprintf("Message sent to %d subscribers.", len(subscriberList))
			}
		}
		renderDefaultTemplate(w, "views/landing.gohtml", templateData)
		return
	}
}

/* This is the shape of the r.Form submitted when MessageBird forwards an SMS as a POST request to a URL.
map[message_id:[7a76afeaef3743d28d0e2d93621235ca] originator:[16132093477] reference:[47749346971] createdDatetime:[2018-09-24T08:30:59+00:00] id:[f91908b75f9e4b1fba3b96dc44995f03] message:[this is a test message] receiver:[14708000894] body:[this is a test message] date:[1537806659] payload:[this is a test message] sender:[16132093477] date_utc:[1537777859] recipient:[14708000894]]
*/

func (app *appContext) updateSubscriptionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			thisNumber := "+" + r.FormValue("receiver")
			subscriber := "+" + r.FormValue("originator")
			payload := r.FormValue("payload")

			// Decide what message to send to subscriber.
			switch {
			case strings.Contains(payload, "SUBSCRIBE"):
				err := app.upsertDB(subscriber, "yes")
				if err != nil {
					log.Println(err)
				}
				err = app.sendSMS("You've been subscribed! Reply \"STOP\" to stop receiving these messages.", "MBSender", []string{subscriber})
				if err != nil {
					log.Println(err)
				}
			case strings.Contains(payload, "STOP"):
				err := app.upsertDB(subscriber, "no")
				if err != nil {
					log.Println(err)
				}
				err = app.sendSMS("You've been unsubscribed! Reply \"SUBSCRIBE\" if you changed your mind!", "MBSender", []string{subscriber})
				if err != nil {
					log.Println(err)
				}
			default:
				err := app.sendSMS("SMS \"SUBSCRIBE\" to this number to subscribe to these messages.", thisNumber, []string{subscriber})
				if err != nil {
					log.Println(err)
				}
			}
			log.Printf("Recieved %s request.", r.Method)
		}
	}
}

func renderDefaultTemplate(w http.ResponseWriter, thisView string, data interface{}) {
	renderthis := []string{thisView, "views/layouts/default.gohtml"}
	t, err := template.ParseFiles(renderthis...)
	if err != nil {
		log.Fatal(err)
	}
	err = t.ExecuteTemplate(w, "default", data)
	if err != nil {
		log.Fatal(err)
	}
}
