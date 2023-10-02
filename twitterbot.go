package main

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	"log"
	"os"
	"strings"
	"time"
)

var lastTweetId int64

func main() {
	// Initialize Mattermost and Twitter clients
	mattermostClient := model.NewAPIv4Client(os.Getenv("TWITTER_BOT_MM_SERVER"))
	mattermostClient.SetToken(os.Getenv("TWITTER_BOT_MM_TOKEN"))

	config := oauth1.NewConfig(os.Getenv("TWITTER_BOT_CONSUMER_KEY"), os.Getenv("TWITTER_BOT_CONSUMER_SECRET"))
	token := oauth1.NewToken(os.Getenv("TWITTER_BOT_ACCESS_TOKEN"), os.Getenv("TWITTER_BOT_ACCESS_SECRET"))
	httpClient := config.Client(oauth1.NoContext, token)
	twitterClient := twitter.NewClient(httpClient)

	// Start WebSocket client for real-time updates
	go startWebSocketClient(mattermostClient)

	// Main loop for fetching and posting tweets
	for {
		fetchAndPostTweets(mattermostClient, twitterClient)
		time.Sleep(5 * time.Minute)
	}
}

func fetchAndPostTweets(mattermostClient *model.Client4, twitterClient *twitter.Client) {
	channelId := os.Getenv("TWITTER_BOT_MM_CHANNEL_ID")

	// Fetch latest tweets
	params := &twitter.HomeTimelineParams{
		Count:   20,
		SinceID: lastTweetId,
	}
	tweets, _, err := twitterClient.Timelines.HomeTimeline(params)
	if err != nil {
		log.Fatal(err)
	}

	// Post tweets to Mattermost
	for _, tweet := range tweets {
		post := &model.Post{
			ChannelId: channelId,
			Message:   tweet.Text,
			Props: map[string]interface{}{
				"tweet_id": tweet.IDStr,
			},
		}
		_, _, _ = mattermostClient.CreatePost(post)

		// Update lastTweetId
		if tweet.ID > lastTweetId {
			lastTweetId = tweet.ID
		}
	}
}

func startWebSocketClient(mattermostClient *model.Client4) {
	server := os.Getenv("TWITTER_BOT_MM_SERVER")
	ws := strings.Replace(server, "http", "ws", 1)
	webSocketClient, err := model.NewWebSocketClient4(ws, mattermostClient.AuthToken)
	if err != nil {
		log.Fatal(err)
	}

	webSocketClient.Listen()

	for event := range webSocketClient.EventChannel {
		handleWebSocketEvent(event, mattermostClient)
	}
}

func handleWebSocketEvent(event *model.WebSocketEvent, mmClient *model.Client4) {
	owner := os.Getenv("TWITTER_BOT_OWNER_USER_ID")

	switch event.EventType() {
	case model.WebsocketEventReactionAdded:
		// handle reactions
		data := event.GetData()
		mlog.Info("reaction added", mlog.Any("data", data), mlog.Any("owner", owner))
		//reaction := model.ReactionFromJson(data)
		//if reaction.UserId == specificUserId {
		//	// Like the tweet
		//	post, _ := mmClient.GetPost(reaction.PostId, "")
		//	tweetIdStr := post.Props["tweet_id"].(string)
		//	tweetId, _ := strconv.ParseInt(tweetIdStr, 10, 64)
		//	// Assuming twitterClient is the initialized Twitter client
		//	// ...
		//	_, _, _ = twitterClient.Favorites.Create(&twitter.FavoriteCreateParams{ID: tweetId})

	case model.WebsocketEventReactionRemoved:
		// Handle reaction removed
		data := event.GetData()
		mlog.Info("reaction removed", mlog.Any("data", data))

		//reaction := model.ReactionFromJson(event.Data["reaction"].(string))
		//if reaction.UserId == owner {
		//	post, _, _ := mmClient.GetPost(reaction.PostId, "")
		//	tweetIdStr := post.Props["tweet_id"].(string)
		//	tweetId, _ := strconv.ParseInt(tweetIdStr, 10, 64)
		//	_, _, _ = twitterClient.Favorites.Destroy(&twitter.FavoriteDestroyParams{ID: tweetId})
		//}

		//case model.WEBSOCKET_EVENT_POSTED:
		//	// handle new posts
		//	post := model.PostFromJson(event.Data["post"].(string))
		//	if post.UserId == specificUserId && post.ParentId != "" {
		//		// Assuming post.ParentId exists in our map or props
		//		parentPost, _ := mmClient.GetPost(post.ParentId, "")
		//		tweetIdStr := parentPost.Props["tweet_id"].(string)
		//		tweetId, _ := strconv.ParseInt(tweetIdStr, 10, 64)
		//		// Post this as a reply to the tweet
		//		// ...
		//		_, _, _ = twitterClient.Statuses.Update(post.Message, &twitter.StatusUpdateParams{InReplyToStatusID: tweetId})
		//	}
	}
}
