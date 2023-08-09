package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/machinebox/graphql"
	"github.com/rs/cors"
)

type InitResponse struct {
	Components []Component `json:"components"`
}

type Canvas struct {
	Content Content `json:"content"`
}

type Content struct {
	Components []Component `json:"components"`
}

type Component struct {
	Type        string   `json:"type"`
	Text        string   `json:"text,omitempty"`
	Style       string   `json:"style,omitempty"`
	Id          string   `json:"id,omitempty"`
	Label       string   `json:"label,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Size        string   `json:"size,omitempty"`
	Action      *Action  `json:"action,omitempty"`
	Options     []Option `json:"options,omitempty"`
}

type Action struct {
	Type string `json:"type"`
}

type Option struct {
	Type string `json:"type"`
	Id   string `json:"id"`
	Text string `json:"text"`
}
type SubmitResponse struct {
	UserAddress string `json:"address"`
	CampaignId  string `json:"campaignId"`
	SpaceId     string `json:"spaceId"`
}

type CampaignQueryResponse struct {
	Campaign CampaignDetails `json:"campaign"`
}

type CampaignDetails struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	Space        Space   `json:"space"`
	NFTCore      NFTCore `json:"nftCore"`
	IsNFTHolder  bool    `json:"isNFTHolder"`
	ClaimedTimes int     `json:"claimedTimes"`
}

type Space struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsVerified bool   `json:"isVerified"`
}

type NFTCore struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	ContractAddress string `json:"contractAddress"`
	Chain           string `json:"chain"`
}

type SpaceQueryResponse struct {
	Space SpaceDetails `json:"space"`
}

type SpaceDetails struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Chains    []string  `json:"chains"`
	Campaigns Campaigns `json:"campaigns"`
}

type Campaigns struct {
	List []CampaignIdList `json:"list"`
}

type CampaignIdList struct {
	ID string `json:"id"`
}

type CampaignComponent struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Style string `json:"style,omitempty"`
}

type Payload struct {
	ConversationID int                    `json:"conversation_id"`
	InboxAppID     int                    `json:"inbox_app_id"`
	AdminID        int                    `json:"admin_id"`
	AppID          string                 `json:"app_id"`
	UserID         string                 `json:"user_id"`
	ComponentID    string                 `json:"component_id"`
	InputValues    SubmitResponse         `json:"input_values"`
	CurrentCanvas  map[string]interface{} `json:"current_canvas"` // Or use a dedicated struct if needed
}

func main() {
	r := mux.NewRouter()
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})

	r.Use(c.Handler)

	r.HandleFunc("/init", InitCanvasKit)
	r.HandleFunc("/submit", Submit)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Listening on " + port + "...")
	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatal(err)
	}
}

func InitCanvasKit(w http.ResponseWriter, r *http.Request) {
	fmt.Println("InitCanvasKit")
	// Construct the response
	canvas := map[string]interface{}{
		"canvas": Canvas{
			Content: Content{
				Components: []Component{
					{
						Type:  "text",
						Text:  "*Check address Galxe nft balance*",
						Style: "header",
					},
					{
						Type: "spacer",
						Size: "s",
					},
					{
						Type:        "input",
						Id:          "address",
						Label:       "User Address",
						Placeholder: "0x...",
					},
					{
						Type: "spacer",
						Size: "s",
					},
					{
						Type:  "input",
						Id:    "campaignId",
						Label: "campaign Id",
					},
					{
						Type:  "text",
						Text:  "*Or*",
						Style: "paragraph",
					},
					{
						Type:  "input",
						Id:    "spaceId",
						Label: "Space Id",
					},
					{
						Type: "spacer",
						Size: "s",
					},
					{
						Type:  "button",
						Id:    "query-address",
						Label: "Check Address Balance",
						Style: "primary",
						Action: &Action{
							Type: "submit",
						},
					},
				},
			},
		},
	}

	// Marshal the response into JSON
	responseJSON, err := json.Marshal(canvas)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

func Submit(w http.ResponseWriter, r *http.Request) {
	var payload Payload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Println("Error decoding request body:", err)
		RenderErrorCanvas(w, err)
		return
	}

	//log body
	log.Println("Submit:", payload)

	res := payload.InputValues

	spaceIdInt := 0
	if res.SpaceId != "" {
		spaceIdInt, err = strconv.Atoi(res.SpaceId)
		if err != nil {
			log.Println("Error converting spaceId to int:", err)
			RenderErrorCanvas(w, err)
			return
		}
	}

	log.Println("SubmitResponse:", res)

	// Check for required fields
	if res.UserAddress == "" {
		log.Println("Error: UserAddress is required")
		RenderErrorCanvas(w, err)
		return
	}

	// Ensure at least one of SpaceId or CampaignId is provided
	if spaceIdInt == 0 && res.CampaignId == "" {
		log.Println("Error: SpaceId or CampaignId is required")
		RenderErrorCanvas(w, err)
		return
	}

	client := graphql.NewClient("https://graphigo.prd.galaxy.eco/query")

	// switch for campaign or space
	if res.CampaignId != "" {
		// query campaign
		campaignInfo, err := QueryCampaign(client, res.CampaignId, res.UserAddress)
		if err != nil {
			log.Println("Error querying campaign:", err) // Enhanced logging here
			RenderErrorCanvas(w, err)
			return
		}

		response := map[string]interface{}{
			"canvas": Canvas{
				Content: Content{
					Components: BuildCampaignComponents([]CampaignQueryResponse{campaignInfo}),
				},
			},
		}

		// Extract components from the constructed response
		components := response["canvas"].(Canvas).Content.Components

		// Append the "Query Again" button
		components = append(components, Component{
			Type:   "button",
			Id:     "query-again",
			Label:  "Query Again",
			Style:  "primary",
			Action: &Action{Type: "submit"},
		})

		// Reconstruct the response with the modified components
		response["canvas"] = Canvas{
			Content: Content{
				Components: components,
			},
		}

		// Marshal the response into JSON
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Println("Error marshalling response:", err) // Enhanced logging here
			RenderErrorCanvas(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(responseJSON)
	} else {
		// query space
		log.Println("Querying space:", spaceIdInt)
		spaceInfo, err := QuerySpace(client, spaceIdInt)
		if err != nil {
			log.Println("Error querying space:", err) // Enhanced logging here
			RenderErrorCanvas(w, err)
			return
		}

		//loop through spaces for campaigns ids and query each campaign
		var campaigns []CampaignQueryResponse
		var mu sync.Mutex // Mutex to protect concurrent writes to the campaigns slice

		var wg sync.WaitGroup // WaitGroup to wait for all goroutines to finish

		for _, campaign := range spaceInfo.Space.Campaigns.List {
			wg.Add(1) // Increment the WaitGroup counter
			go func(campaignID string) {
				defer wg.Done() // Decrement the WaitGroup counter when done
				campaignInfo, err := QueryCampaign(client, campaignID, res.UserAddress)
				if err != nil {
					log.Println("Error querying campaign:", err) // Enhanced logging here
					RenderErrorCanvas(w, err)
					return
				}
				mu.Lock() // Lock the mutex
				campaigns = append(campaigns, campaignInfo)
				mu.Unlock() // Unlock the mutex
			}(campaign.ID)
		}

		wg.Wait() // Wait for all goroutines to finish

		response := map[string]interface{}{
			"canvas": Canvas{
				Content: Content{
					Components: BuildCampaignComponents(campaigns),
				},
			},
		}
		// Extract components from the constructed response
		components := response["canvas"].(Canvas).Content.Components

		// Append the "Query Again" button
		components = append(components, Component{
			Type:   "button",
			Id:     "query-again",
			Label:  "Query Again",
			Style:  "primary",
			Action: &Action{Type: "submit"},
		})

		// Reconstruct the response with the modified components
		response["canvas"] = Canvas{
			Content: Content{
				Components: components,
			},
		}
		// Marshal the response into JSON
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Println("Error marshalling response:", err) // Enhanced logging here
			RenderErrorCanvas(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(responseJSON)
	}
}

func QueryCampaign(client *graphql.Client, id string, address string) (CampaignQueryResponse, error) {
	query := fmt.Sprintf(`
	{
		campaign(id: "%s") {
			id
			name
			status
			space {
				id
				name
				isVerified
			}
			nftCore {
				id
				name
				symbol
				contractAddress
				chain
			}
			isNFTHolder(address: "%s")
			claimedTimes(address: "%s")
		}
	}
	`, id, address, address)

	req := graphql.NewRequest(query)

	ctx := context.Background()

	var respData CampaignQueryResponse
	if err := client.Run(ctx, req, &respData); err != nil {
		log.Println("Error querying campaign:", err)
		return respData, err
	}
	return respData, nil
}

func QuerySpace(client *graphql.Client, id int) (SpaceQueryResponse, error) {
	queryString := fmt.Sprintf(`
	query {
		space(id: %d) {
		  id
		  name
		  campaigns(input: { spaceId: %d }) {
			totalCount
			list {
				id
			}
		  }
		}
	  }
	`, id, id)
	req := graphql.NewRequest(queryString)
	ctx := context.Background()

	var respData SpaceQueryResponse
	if err := client.Run(ctx, req, &respData); err != nil {
		log.Println("Error querying space:", err)
		return respData, err
	}
	return respData, nil
}

func BuildCampaignComponents(campaigns []CampaignQueryResponse) []Component {
	var components []Component

	for _, campaign := range campaigns {
		// Add campaign ID
		components = append(components, Component{
			Type:  "text",
			Text:  fmt.Sprintf("Campaign ID: %s", campaign.Campaign.ID),
			Style: "header",
		})

		// Add campaign Name
		components = append(components, Component{
			Type:  "text",
			Text:  fmt.Sprintf("Name: %s", campaign.Campaign.Name),
			Style: "paragraph",
		})

		// Add NFT Holder status
		components = append(components, Component{
			Type:  "text",
			Text:  fmt.Sprintf("Is NFT Holder: %v", campaign.Campaign.IsNFTHolder),
			Style: "paragraph",
		})

		// Add Claimed Times
		components = append(components, Component{
			Type:  "text",
			Text:  fmt.Sprintf("Claimed Times: %d", campaign.Campaign.ClaimedTimes),
			Style: "paragraph",
		})

		// // Add a spacer for better visual separation between campaigns
		// components = append(components, Component{
		// 	Type: "spacer",
		// 	Size: "s",
		// })
	}

	return components
}

func BuildErrorComponents(err error) []Component {
	var components []Component

	// Check if the error is nil
	errorMsg := "Unknown error occurred"
	if err != nil {
		errorMsg = err.Error()
	}

	components = append(components, Component{
		Type:  "text",
		Text:  fmt.Sprintf("Error: %s", errorMsg),
		Style: "header",
	})

	// Add a spacer for better visual separation
	components = append(components, Component{
		Type: "spacer",
		Size: "s",
	})

	// Add the "Refresh" button
	components = append(components, Component{
		Type:  "button",
		Id:    "refresh-button",
		Label: "Refresh",
		Style: "primary",
		Action: &Action{
			Type: "init",
		},
	})

	return components
}

func RenderErrorCanvas(w http.ResponseWriter, err error) {
	response := map[string]interface{}{
		"canvas": Canvas{
			Content: Content{
				Components: BuildErrorComponents(err),
			},
		},
	}

	responseJSON, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		http.Error(w, "Failed to marshal error response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}
