package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const mp4ContentType = "video/mp4"
const twitterUserPrefix = "https://x.com/"

type TwitterService struct {
	Service     Type
	urlRegexp   *regexp.Regexp
	bearerToken string
	authToken   string
	client      *http.Client
}

func NewTwitterService() *TwitterService {
	bearerToken := viper.GetString("twitter.bearer_token")
	authToken := viper.GetString("twitter.auth_token")

	return &TwitterService{
		Service:     Twitter,
		urlRegexp:   regexp.MustCompile(`(?i)https?:\/\/(?:(?:www|mobile)\.)?(?:vx|fx|fixup)?(?:twitter|x)\.com\/(.+?)\/status\/(\d+)`),
		bearerToken: bearerToken,
		authToken:   authToken,
		client:      &http.Client{},
	}
}

func (s TwitterService) CheckValid(urlString string) (*IncomingURL, bool) {
	match := s.urlRegexp.FindStringSubmatch(urlString)

	if match == nil {
		return nil, false
	}

	strID := match[2]
	intID, _ := strconv.Atoi(strID)
	url := fmt.Sprintf("%s%s/status/%s", twitterUserPrefix, match[1], match[2])

	return &IncomingURL{
		Service:  s.Service,
		Original: urlString,
		URL:      url,
		StrID:    strID,
		IntID:    intID,
	}, true
}

func (s TwitterService) IsService(serviceType Type) bool {
	return serviceType == s.Service
}

type TweetResponse struct {
	Data struct {
		ThreadedConversationWithInjectionsV2 struct {
			Instructions []json.RawMessage
		} `json:"threaded_conversation_with_injections_v2"`
	}
}

type TimelineAddEntries struct {
	Entries []json.RawMessage
}

type TweetCore struct {
	UserResults struct {
		Result struct {
			Legacy struct {
				Name       string
				ScreenName string `json:"screen_name"`
			}
		}
	} `json:"user_results"`
}

type TweetLegacy struct {
	FullText         string `json:"full_text"`
	DisplayTextRange []int  `json:"display_text_range"`
	Entities         struct {
		Media []EntityMedia
	}
	ExtendedEntities struct {
		Media []EntityMedia
	} `json:"extended_entities"`
}

type TweetEntity struct {
	Content struct {
		ItemContent struct {
			TweetResults struct {
				Result struct {
					TypeName string `json:"__typename"`
					Core     TweetCore
					Legacy   TweetLegacy
					Tweet    struct {
						Core   TweetCore
						Legacy TweetLegacy
					}
				}
			} `json:"tweet_results"`
		}
	}
}

type EntityMedia struct {
	Type          string
	MediaUrlHttps string `json:"media_url_https"`
	VideoInfo     struct {
		Variants []struct {
			ContentType string `json:"content_type"`
			Url         string
		}
	} `json:"video_info"`
}

func (s TwitterService) ExtractMediaFromURL(incomingURL *IncomingURL) ([]*Media, error) {
	var result []*Media

	req, err := http.NewRequest("GET", "https://x.com/i/api/graphql/xd_EMdYvB9hfZsZ6Idri0w/TweetDetail", nil)
	if err != nil {
		return result, err
	}

	csrfToken := strings.Replace(uuid.New().String(), "-", "", -1)

	req.Header.Set("Authorization", "Bearer "+s.bearerToken)
	req.Header.Set("Cookie", fmt.Sprintf("ct0=%s; auth_token=%s", csrfToken, s.authToken))
	req.Header.Set("x-csrf-token", csrfToken)
	req.Header.Set("x-twitter-auth-type", "OAuth2Session")
	req.Header.Set("x-twitter-client-language", "en")
	req.Header.Set("x-twitter-active-user", "yes")
	req.Header.Set("x-client-transaction-id", "RiYhUS2woF9cK5mEtB5te+NZ6wnqJwGn7OrvSvqIp1Aug/LKhd15QQBoc1czNq0M4BpQjEWHdj1DcNR1SmTjZW9jLQNhRQ")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0")

	q := req.URL.Query()
	q.Add("features", (`{"rweb_video_screen_enabled":false,"profile_label_improvements_pcf_label_in_post_enabled":true,"rweb_tipjar_consumption_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"premium_content_api_read_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"responsive_web_grok_analyze_button_fetch_trends_enabled":false,"responsive_web_grok_analyze_post_followups_enabled":true,"responsive_web_jetfuel_frame":false,"responsive_web_grok_share_attachment_enabled":true,"articles_preview_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"responsive_web_grok_show_grok_translated_post":false,"responsive_web_grok_analysis_button_from_backend":true,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_grok_image_annotation_enabled":true,"responsive_web_enhance_cards_enabled":false}`))
	q.Add("fieldToggles", (`{"withArticleRichContentState":true,"withArticlePlainText":false,"withGrokAnalyze":false,"withDisallowedReplyControls":false}`))
	q.Add("variables", fmt.Sprintf(`{"focalTweetId":"%s","referrer":"tweet","with_rux_injections":false,"rankingMode":"Relevance","includePromotedContent":true,"withCommunity":true,"withQuickPromoteEligibilityTweetFields":true,"withBirdwatchNotes":true,"withVoice":true}`, incomingURL.StrID))
	req.URL.RawQuery = q.Encode()

	client := s.client

	resp, err := client.Do(req)

	if err != nil {
		log.WithError(err).Error("failed to send request")
		return result, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Error("failed to read response body")
		return result, err
	}

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
			"body":        string(body),
		}).Error("failed to get tweet detail")
		return result, errors.New("failed to get tweet detail")
	}

	var tweetResponse TweetResponse
	err = json.Unmarshal(body, &tweetResponse)
	if err != nil {
		log.WithError(err).Error("failed to unmarshal response body")
		return result, err
	}

	var timelineAddEntries TimelineAddEntries
	err = json.Unmarshal(tweetResponse.Data.ThreadedConversationWithInjectionsV2.Instructions[0], &timelineAddEntries)
	if err != nil {
		return result, errors.New("can't parse TimelineAddEntries")
	}

	var tweetEntity TweetEntity
	err = json.Unmarshal(timelineAddEntries.Entries[0], &tweetEntity)
	if err != nil {
		return result, errors.New("can't parse TweetEntity")
	}

	var tweetLegacy TweetLegacy
	var tweetCore TweetCore
	if tweetEntity.Content.ItemContent.TweetResults.Result.TypeName == "Tweet" {
		tweetLegacy = tweetEntity.Content.ItemContent.TweetResults.Result.Legacy
		tweetCore = tweetEntity.Content.ItemContent.TweetResults.Result.Core
	} else {
		tweetLegacy = tweetEntity.Content.ItemContent.TweetResults.Result.Tweet.Legacy
		tweetCore = tweetEntity.Content.ItemContent.TweetResults.Result.Tweet.Core
	}

	mediaEntities := tweetLegacy.Entities.Media
	extendedMediaEntities := tweetLegacy.ExtendedEntities.Media

	if len(extendedMediaEntities) >= len(mediaEntities) {
		mediaEntities = extendedMediaEntities
	}

	for _, mediaEntity := range mediaEntities {
		var resultMedia *Media

		switch mediaEntity.Type {
		case "photo":
			resultMedia = s.extractPhoto(&mediaEntity)
		case "animated_gif":
			fallthrough
		case "video":
			resultMedia = s.extractVideo(&mediaEntity)
		default:
			continue
		}

		resultMedia.Service = string(s.Service)
		resultMedia.Source = incomingURL.URL
		resultMedia.FileName = fmt.Sprintf("@%s_%s", tweetEntity.Content.ItemContent.TweetResults.Result.Tweet.Core.UserResults.Result.Legacy.ScreenName, resultMedia.FileName)
		s.completeMediaMeta(resultMedia, &tweetLegacy, &tweetCore)

		result = append(result, resultMedia)
	}

	return result, nil
}

func (s TwitterService) completeMediaMeta(media *Media, tweetLegacy *TweetLegacy, tweetCore *TweetCore) {
	media.Author = tweetCore.UserResults.Result.Legacy.Name
	media.AuthorURL = twitterUserPrefix + tweetCore.UserResults.Result.Legacy.ScreenName
	media.Description = string([]rune(tweetLegacy.FullText)[tweetLegacy.DisplayTextRange[0]:tweetLegacy.DisplayTextRange[1]])
}

func (s TwitterService) extractPhoto(media *EntityMedia) *Media {
	urlParts := strings.Split(media.MediaUrlHttps, "/")
	// wxt2005_1.jpg
	fileName := urlParts[len(urlParts)-1]

	return &Media{
		FileName: fileName,
		URL:      media.MediaUrlHttps,
		Type:     "photo",
	}
}

func (s TwitterService) extractVideo(media *EntityMedia) *Media {
	variants := media.VideoInfo.Variants
	videoUrl := ""
	for _, item := range variants {
		if item.ContentType == mp4ContentType {
			videoUrl = item.Url
		}
	}

	if videoUrl == "" {
		return nil
	}

	// videoUrl := variant.Url
	urlParts := strings.Split(videoUrl, "/")
	// wxt2005_1.mp4
	fileName := urlParts[len(urlParts)-1]

	return &Media{
		FileName: fileName,
		URL:      videoUrl,
		Type:     "video",
	}
}
