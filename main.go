package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
	"github.com/machinebox/graphql"
	"github.com/rs/cors"
)

var InitialCanvasKit = []Component{
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
		Id:    "spaceId",
		Label: "Space Id",
	},
	{
		Type:  "text",
		Text:  "*Or*",
		Style: "paragraph",
	},
	{
		Type:  "input",
		Id:    "campaignId",
		Label: "campaign Id",
	},
	{
		Type: "spacer",
		Size: "s",
	},
	{
		Type:   "button",
		Id:     "query-address",
		Label:  "Check Address Balance",
		Style:  "primary",
		Action: Action{Type: "submit"},
	},
}

type InitResponse struct {
	Components []Component `json:"components"`
}

type CanvasContent struct {
	Content ContentDetails `json:"content"`
}

type CanvasResponse struct {
	Canvas CanvasContent `json:"canvas"`
}

type ContentDetails struct {
	Components []Component `json:"components"`
}

type Component struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Style       string `json:"style,omitempty"`
	Id          string `json:"id,omitempty"`
	Label       string `json:"label,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Size        string `json:"size,omitempty"`
	Action      Action `json:"action,omitempty"`
}

type Action struct {
	Type string `json:"type,omitempty"`
}

type SubmitResponse struct {
	UserAddress string `json:"userAddress"`
	CampaignId  string `json:"campaignId"`
	SpaceId     int    `json:"spaceId"`
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
	response := CanvasResponse{
		Canvas: CanvasContent{
			Content: ContentDetails{
				Components: InitialCanvasKit,
			},
		},
	}

	// Marshal the response into JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

func Submit(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Submit")

	// read req body
	var res SubmitResponse
	err := json.NewDecoder(r.Body).Decode(&res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check for required fields
	if res.UserAddress == "" {
		http.Error(w, "UserAddress is required", http.StatusBadRequest)
		return
	}

	// Ensure at least one of SpaceId or CampaignId is provided
	if res.SpaceId == 0 && res.CampaignId == "" {
		http.Error(w, "Either SpaceId or CampaignId must be provided", http.StatusBadRequest)
		return
	}

	client := graphql.NewClient("https://graphigo.prd.galaxy.eco/query")

	// switch for campaign or space
	if res.CampaignId != "" {
		// query campaign
		campaignInfo, err := QueryCampaign(client, res.CampaignId, res.UserAddress)
		if err != nil {
			log.Println("Error querying campaign:", err) // Enhanced logging here
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := json.Marshal(campaignInfo)
		if err != nil {
			log.Println("Error marshalling campaign:", err) // Enhanced logging here
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(res)

	} else {
		// query space
		log.Println("Querying space:", res.SpaceId)
		spaceInfo, err := QuerySpace(client, res.SpaceId, res.UserAddress)
		if err != nil {
			log.Println("Error querying space:", err) // Enhanced logging here
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
					// You might want to handle the error better here
					return
				}
				mu.Lock() // Lock the mutex
				campaigns = append(campaigns, campaignInfo)
				mu.Unlock() // Unlock the mutex
			}(campaign.ID)
		}

		wg.Wait() // Wait for all goroutines to finish

		campaignComponents := BuildCampaignComponents(campaigns)

		// Convert components to JSON and send the response
		res, err := json.Marshal(campaignComponents)
		if err != nil {
			log.Println("Error marshalling campaign components:", err) // Enhanced logging here
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
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

func QuerySpace(client *graphql.Client, id int, address string) (SpaceQueryResponse, error) {
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

func BuildCampaignComponents(campaigns []CampaignQueryResponse) []CampaignComponent {
	var components []CampaignComponent

	for _, campaign := range campaigns {
		component := CampaignComponent{
			Type:  "text",
			Text:  fmt.Sprintf("Campaign ID: %s, Name: %s, IsNftHolder: %v, ClaimedTimes: %d ", campaign.Campaign.ID, campaign.Campaign.Name, campaign.Campaign.IsNFTHolder, campaign.Campaign.ClaimedTimes),
			Style: "paragraph",
		}
		components = append(components, component)
	}

	return components
}
