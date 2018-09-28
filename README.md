# SMS Marketing Subscriptions
### ⏱ 30 min build time 

## Why build SMS marketing subscriptions? 

SMS makes it incredibly easy for businesses to reach consumers everywhere at any time, directly on their mobile devices. For many people, these messages are a great way to discover things like discounts and special offers from a company, while others might find them annoying. For this reason, it is  important and also required by law in many countries, to provide clear opt-in and opt-out mechanisms for SMS broadcast lists. To make these work independently of a website it's useful to assign a programmable [virtual mobile number](https://www.messagebird.com/en/numbers) (VMN) to your SMS campaign and handle incoming messages programmatically so users can control their subscription with basic command keywords.

In this MessageBird Developer Guide, we'll show you how to implement an SMS marketing campaign subscription tool built as a sample application in Node.js.

This application implements the following:
* A person can send the keyword _SUBSCRIBE_ to a specific VMN, that the company includes in their advertising material, to opt in to messages, which is immediately confirmed.
* If the person no longer wants to receive messages they can send the keyword _STOP_ to the same number. Opt-out is also confirmed.
* An administrator can enter a message in a form on a website. Then they can send this message to all confirmed subscribers immediately.

## Getting Started

Before we get started, let's first make sure that you've installed the following:

- Go 1.11 and newer.
- [MessageBird Go SDK](https://github.com/messagebird/go-rest-api) 5.0.0 and newer.

Now, let's install the MessageBird Go SDK with the `go get` command:

```bash
go get -u -v github.com/messagebird/go-rest-api
```

To get started, we need to:

- [Configure the MessageBird SDK](#configuring-the-messagebird-sdk)
- Set up our MessageBird account with the [prerequisites for receiving messages](#prerequisites-for-receiving-messages)

You'll find the sample code for this guide at the [MessageBird Developer Guides GitHub repository](https://github.com/messagebirdguides/subscriptions-guide-go). Download or clone the repository to view the example code and follow along with the guide.

## Configuring the MessageBird SDK

While the MessageBird SDK and an API key are not required to receive messages, it is necessary for sending confirmations and our marketing messages.

Get your MessageBird API key from the [API access (REST) tab](https://dashboard.messagebird.com/en/developers/access) in the _Developers_ section of your MessageBird account. You'll need to add this to your application to send SMS messages to your subscribers.

To keep this guide straightforward, we'll be hardcoding the MessageBird API key in our application. But for production-ready applications, you'll want to store your API key in a configuration file and access it using a library like [GoDotEnv](https://github.com/joho/godotenv).

To initialize the MessageBird SDK in your Go application, add the following line of code:

```go
client := messagebird.New("<enter-your-api-key>")
```

We'll get into where we'll need to do this later in the guide as we start writing our application.

## Prerequisites for Receiving Messages

### Overview

This guide describes receiving messages using MessageBird. From a high-level viewpoint, receiving is relatively simple: your application defines a _webhook URL_, which you assign to a number purchased on the MessageBird Dashboard using [Flow Builder](https://dashboard.messagebird.com/en/flow-builder). Whenever someone sends a message to that number, MessageBird collects it and forwards it to the webhook URL, where you can process it.

### Exposing your Development Server with localtunnel

One small roadblock when working with webhooks is the fact that MessageBird needs to access your application, so it needs to be available on a public URL. During development, you're typically working in a local development environment that is not publicly available. Thankfully this is not a big deal since various tools and services allow you to quickly expose your development environment to the Internet by providing a tunnel from a public URL to your local machine. One of these tools is [localtunnel.me](https://localtunnel.me), which you can install using npm:

````bash
npm install -g localtunnel
````

Start a tunnel by providing a local port number on which your application runs. Because sample is configured to run on port 8080, launch your tunnel with this command:

````bash
lt --port 8080
````

After you've launched the tunnel, localtunnel displays your temporary public URL. We'll need that in a minute.

Another common tool for tunneling your local machine is [ngrok](https://ngrok.com), which you can have a look at if you're facing problems with localtunnel.me.

### Get an Inbound Number

An obvious requirement for receiving messages is an inbound number. Virtual mobile numbers look and work similar to regular mobile numbers, however, instead of being attached to a mobile device via a SIM card, they live in the cloud, i.e., a data center, and can process incoming SMS and voice calls. Explore our low-cost programmable and configurable numbers [here](https://www.messagebird.com/en/numbers).

Here's how to purchase one:

1. Go to the [Numbers](https://dashboard.messagebird.com/en/numbers) section of your MessageBird account and click **Buy a number**.
2. Choose the country in which you and your customers are located and make sure the _SMS_ capability is selected.
3. Choose one number from the selection and the duration for which you want to prepay the amount. ![Buy a number screenshot](/assets/images/screenshots/buy-a-number.png)
4. Confirm by clicking **Buy Number**.

Congratulations, you have set up your first virtual mobile number!

### Connect Number to the Webhook

So you have a number now, but MessageBird has no idea what to do with it. That's why you need to define a _Flow_ next that links your number to your webhook. This is how you do it:

1. On the [Numbers](https://dashboard.messagebird.com/en/numbers) section of your MessageBird account, click the "add new flow" icon next to the number you purchased in the previous step. ![Create Flow, Step 1](/assets/images/screenshots/.png)
2. Choose **Incoming SMS** as the trigger event. ![Create Flow, Step 2](/assets/images/screenshots/create-flow-2.png)
3. Click the small **+** to add a new step to your flow and choose **Forward to URL**. ![Create Flow, Step 3](/assets/images/screenshots/create-flow-3.png)
4. Choose _POST_ as the method, copy the output from the `lt` command in the previous stop and add `/webhook` to it - this is the name of the route we use to handle incoming messages in our sample application. Click **Save**. ![Create Flow, Step 4](/assets/images/screenshots/create-flow-4.png)
5. Hit **Publish Changes** and your flow becomes active! Well done, another step closer to testing incoming messages!

If you have more than one flow, it might be useful to rename it this flow, because _Untitled flow_ won't be helpful in the long run. You can do that by editing the flow and clicking the three dots next to the name and choose **Edit flow name**.

**NOTE**: A number must be added to your MessageBird contact list before you can send SMS messages to it. To make sure that any phone number that sends a message to your VMN can receive messages from you, you can add an **Add Contact** step just before the **Forward to URL** step. Be sure to configure the **Add Contact** step to add the "sender" to your contact list.

## Writing Our Application

Now that we've installed the MessageBird Go SDK, got our API key, and set up our MessageBird account with a VMN and attached a flow to it, we can begin writing our application.

We need to have our application:

- **[Get and update a list of subscribers](#get-and-update-a-list-of-subscribers)**: We need to be able to get a list of subscribers, check if they have consented to receive our marketing broadcast, and update their subscription status.
- **[Set up a web server to receive requests](#set-up-a-web-server-to-receive-requests)**: We need a web server for us to set up a landing page where we can send out our marketing broadcast, and to receive POST requests that the MessageBird server forwards to us.
- **[Receive messaages](#receive-messages)**: Our application must be able to receive POST requests from our MessageBird VMN flow, and react according to the contents of that request.
- **[Send messages](#send-messages)**: Our application must be able to send out messages. We have two types of messages that we want to send out: (a) An SMS marketing broadcast to all the subscribers who have opted into our broadcast list; (b) An SMS message that confirms that we have received a request to either subscribe or unsubscribe to our broadcast list.

We also need where and how we're getting our list of subscribers. For this guide, we're using a list of subscribers loaded from a CSV file. In your production-ready application, you would have to replace this data source with an actual list of subscribers drawn from a database or similar.

### Get and Update a List of Subscribers

In the sample code provided, our list of subscribers is stored in subscriberlist.csv. We'll need to load the CSV file into a map, and share it with the rest of the application. We'll do this by writing a `appContext` struct type that we will use to store the map we load from our CSV file and attach HTTP handle methods that we will use for our web server routes.

In your project root, create a file named `main.go`, and enter the following code:

```go
package main

import (
    "log"
)

type appContext struct{
    db *map[string]string
}

func main(){
    db := make(map[string]string)
    app := &appContext{&db}
    err := app.initDB("subscriberlist.csv")
    if err != nil {
        log.Println(err)
    }
}

func (app *appContext) initDB(file string) error{
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
```

Once we call `app.initDB("subscriberlist.csv")`, our application loads the "subscriberlist.csv" file and stores it in `app.db`.

Let's add two methods to help us work with our subscriber list:

```go
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
```

Here, we've added a `getSubscriberList()` method that gives us a list of subscribers who have consented to receive our marketing broadcast, and `upsertDB()` method that updates the status of our subscribers in `app.db`, or adds a new subscriber if they don't exist in `app.db`.

Now that we've set up how we interact with our list of subscribers, we can move on to setting up our web server.

### Set Up a Web Server to Receive Requests

We need to set up a simple web server to serve a landing page that allows us to send a marketing broadcast, and to receive POST requests that the MessageBird server forwards to us from our VMN.

We won't get into the details of how to work with template rendering here; see the `views` folder on how we've got templating set up. We'll just cover how to write our HTTP server and handlers.

First, let's set up our template rendering helper. Add the following code to the bottom of your `main.go` file:

```go
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
```

This loads our `default.gohtml` template, along a single template that we pass to it, and any data we want to render along with the template. For example, we can render our landing template with `renderDefaultTemplate(w, "views/landing.gohtml", &struct{Message string}{"this is a message"})`.

Once we've done that, we can initialize our HTTP server by adding the following code to the bottom of our `main()` block:

```go
func main(){
    // ...

    http.Handle("/", app.landingHandler())
    http.Handle("/webhook", app.updateSubscriptionHandler())

    port := ":8080"
    log.Println("Serving on " + port)
    err = http.ListenAndServe(port, nil)
    if err != nil {
        log.Fatalln(err)
    }

}
```

Here, we've defined two routes:

- `/`, which is our default route and where we will render our landing page.
- `/webhook`, which we will receive POST requests forwarded from our VMN.

Notice that our handlers, `app.landingHandler()` and `app.updateSubscriptionHandler()`, are attached to `app *appContext` as methods. This way, they can access our subscriber list that we've stored in `app.db`.

We'll define our `app.updateSubscriptionHandler()` we we work on [receiving messages](#receive-messages). Right now, we'll define `app.landingHandler()`:

```go
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
```

Here, we're getting our subscriber list and storing it as `subscriberList`, then rendering our `landing.gohtml` view. When we submit a message to broadcast to our subscriber list, we handle it by checking if `r.Method == "POST"`, and then parsing the form submitted for the information we need to send the SMS broadcast.

Notice that we've defined a `templateData` struct at the start of our handler, and that we're using it to pass data to render in `renderDefaultTemplate()` call at the end of our returned `http.HandlerFunc`. This way, we only have to call `renderDefaultTemplate()` once, and change the content we render by updating the `templateData` struct.

With this approach, we can display errors in our rendered template by writing to the `templateData` struct instead of loggint hem to the terminal. For example, instead of `log.Println(err)`, we display errors in our code sample above with `templateData.Message = fmt.Sprintf("Could not send message: %v", err)` and then calling `renderDefaultTemplate(/*..*/,templateData)`.

We've also written a `sendSMS()` helper, which we'll cover in the [sending messages](#sending-messages) section.

Now, we can move on to writing our `app.updateSubscriptionHandler()` function.

### Receive Messages

When MessageBird forwards messages sent to our VMN to our application, it sends a POST request to the URL that we specify in the **Forward to URL** step of our defined flow: `<prefix>.localtunnel.me/webhook` (if you're using localtunnel.me). When we parse the data sent with the POST request, we get a map that looks like the following:

```go
map[message_id:[7a76afeaef3743d28d0e2d93621235ca] originator:[<sender_phone_number>] reference:[47749346971] createdDatetime:[2018-09-24T08:30:59+00:00] id:[f91908b75f9e4b1fba3b96dc44995f03] message:[<message_body>] receiver:[<your_VMN>] body:[<message_body>] date:[1537806659] payload:[<message_body>] sender:[<sender_phone_number>] date_utc:[1537777859] recipient:[<your_VMN>]]
```

Knowing this, let's start writing our `app.updateSubscriptionHandler()` handler. From the POST request, we need to only get the values of the "receiver", originator" and the "payload" keys:

```go
func (app *appContext) updateSubscriptionHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "POST" {
            r.ParseForm()
            thisNumber := "+" + r.FormValue("receiver")
            subscriber := "+" + r.FormValue("originator")
            payload := r.FormValue("payload")
        }
    }
}
```

**Note**: We need to prefix the "receiver" and "originator" values with a "+" to make sure that the phone numbers we pass into our `sms.Create()` call are in the international format.

Then, we need to write a switch statement that handles the following cases:

1. If the received message body contains the string "SUBSCRIBE", add the "originator" to our subscriber list. If the "originator" already exists in our subscriber list, set their subscription status to "yes" to indicate consent to receive our marketing broadcast.
2. If the received message contains the string "STOP", set subscription status of the "originator" to "no" to indicate a withdrawal of consent.
3. If the received message contains neither "SUBSCRIBE" nor "STOP", then do nothing with the subscription list.

Modify `updateSubscriptionHandler()` to look like the following:

```go
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
```

Notice that we're not rendering a view here, because we don't want a publicly available user interface for this. Instead, we'll log all output to the terminal for our own use.

### Sending Messages

Now that we've got most of our application set up, we can get into where we need to send SMS messages and to whom. We need to send SMS messages when:

1. We click **Send to all subscribers** on our rendered landing page to send a marketing broadcast to all numbers on our subscriber list.
2. When we add or remove a subscriber to our list, we should send a confirmation SMS message to the subscriber.
3. When we receive an SMS message which is neither a request to add nor remove the subscriber from our list, we should send an SMS message with a helpful message telling them what keywords their SMS message should contain so that our application can parse their request correctly.

**IMPORTANT**: Make sure that all recipients in your subscriber list have been added to your contact list. If MessageBird fails to send an SMS message to any one recipient in the recipient list, the remaining SMS messages will successfully send but no error will be returned for the SMS messages that fail to send.

We've already added `app.sendSMS()` calls where we need to send SMS messages in our code snippets above, but we haven't defined it yet. Add the following code block just below your `main()` block:

```go
func (app *appContext) sendSMS(msgBody string, originator string, recipients []string) error {
    client := messagebird.New("<enter-your-api-key>")
    msg, err := sms.Create(client, originator, recipients, msgBody, nil)
    if err != nil {
        var mbErrors[]string
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
```

Our error handling is a bit more complex here because we have to read the JSON response from the MessageBird REST API and format it as a single error before returning it.

Once you've done all that, you're ready to run your application. You're done!

### Testing the Application

Double-check that you have set up your number correctly with a flow that forwards incoming messages to a localtunnel URL and that the tunnel is still running. You can restart the tunnel with the `lt` command, but this will change your URL, so you have to update the flow as well.

To start the sample application you have to enter another command, but your existing terminal window is now busy running your tunnel. Therefore you need to open another one. On a Mac you can press _Command_ + _Tab_ to open a second tab that's already pointed to the correct directory. With other operating systems you may have to resort to manually open another terminal window. Once you've got a command prompt, type the following to start the application:

````bash
go run main.go
````

While keeping the terminal open, take out your phone, launch the SMS app and send a message to your virtual mobile number with the keyword "SUBSCRIBE". If you are using a live API key, you should receive a confirmation message stating that you've been subscribed to the broadcast list. Point your browser to http://localhost:8080/ (or your tunnel URL) and you should also see that there's one subscriber. Try sending yourself a message now. And voilá, your marketing system is ready!

## Nice work!

You can adapt the sample application for production by replying mongo-mock with a real MongoDB client, deploying the application to a server and providing that server's URL to your flow. Of course, you should add some authorization to the web form. Otherwise, anybody could send messages to your subscribers.

Don't forget to download the code from the [MessageBird Developer Guides GitHub repository](https://github.com/messagebirdguides/subscriptions-guide-go).

## Next steps

Want to build something similar but not quite sure how to get started? Please feel free to let us know at support@messagebird.com, we'd love to help!
