# Galaxy NFT Balance Checker

This project provides an interface to query and display the balance of Galaxy NFTs for a given user address. It offers endpoints to initialize the Canvas Kit UI and handle submissions.

## Table of Contents

- [Galaxy NFT Balance Checker](#galaxy-nft-balance-checker)
  - [Table of Contents](#table-of-contents)
  - [Installation and Setup](#installation-and-setup)
  - [Usage](#usage)
  - [API Endpoints](#api-endpoints)
  - [Data Structures](#data-structures)
  - [Error Handling](#error-handling)
  - [Concurrent Processing](#concurrent-processing)
  - [Contribution](#contribution)

## Installation and Setup

Before running the project, ensure you have the required dependencies installed. The project mainly uses `mux`, `cors`, and `graphql`.

1. Clone the repository.
2. Navigate to the project directory.
3. Install dependencies:

\```bash
go get -u github.com/gorilla/mux
go get -u github.com/rs/cors
go get -u github.com/machinebox/graphql
\```

4. Run the project:

\```bash
go run main.go
\```

## Usage

The application will start on port `8080` by default. You can set a different port by defining the `PORT` environment variable.

## API Endpoints

1. **/init**  
   Initializes the Canvas Kit with input fields for the user address, campaign ID, and space ID.

2. **/submit**  
   Handles the submission from the Canvas Kit. It processes the user's input and fetches the relevant data from the Galaxy GraphQL endpoint.

## Data Structures

The project makes use of various structs to model the data:

1. **CampaignQueryResponse** and **SpaceQueryResponse**:  
   Structures to capture responses from GraphQL queries.
   
2. **Component**:  
   Represents a UI component in Canvas Kit. Components like text, input, button, etc., can be defined with various attributes.

## Error Handling

Errors are captured and returned to the Canvas Kit interface. When an error occurs, a new Canvas is rendered with the error message and a button to refresh the interface.

## Concurrent Processing

When querying space data, the application fetches details for each campaign in the space concurrently using goroutines. This ensures faster processing and response times.

## Contribution

Feel free to contribute to this project by creating issues or pull requests. Ensure you follow the existing code style and provide detailed commit messages.

