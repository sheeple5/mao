# ⚠️🚨⚠️ WARNING ⚠️🚨⚠️
This is an experimental game I made for fun and uses AI generated code at runtime. There is a high likelihood that this is extremely prompt injectable to RCE, so do yourself a favor and run it in a container.

# Mao
Mao is a card game I learned to play in middle school with my friends. The premise is that it's basically Uno with a regular deck of cards, except with a twist: every time someone wins a game, they get to add a new rule - *one that they don't tell anyone*.
If anyone breaks your rule, they have to draw a card. For anyone who has played this game, it can be diabolical to add a rule that confuses *everyone*, and equally exciting when you have to put on the detective hat to figure out a rule someone else added.
Recently, this game was brought up on [Hacker News](https://news.ycombinator.com/item?id=46611823) and brought me back to what fun it was to play. Inspired, I tried to recreate the experience in Go.

## Description
This implementation of Mao was (for better or for worse) hand written in `go 1.25.5` and intended to be a terminal experience 
(a.k.a I don't have the skills to make an actual application UI. Maybe one day when I gain said skills, I'll circle back to this project and spice it up). There are two `.go` files that make it up: `client.go` and `server.go`.

### server.go
`server.go` is the brains of the operation and acts as, you guessed it, the server for the game. <sub>~how innovative~</sub>

It handles incoming client connections as goroutines and provides a multitude of actions. 
 - healthCheck: Simple health check action to see if the service is up.
 - getStats: Gets the current game standings and sends the client each player's number of wins given a room code.
 - createRoom: Creates a room based on the settings the client has sent the server. It also creates a new player and sends the player data back to the client.
 - joinRoom: Creates a new player in the room if a client requsts to join.
 - leaveRoom: Removes a player from the room if they request to leave.
 - getRooms: Gets the list of public rooms and sends it back to the client.
 - waitStart: Holds a connection open for a client waiting for the game to start and returns their initial hand once it does.
 - startGame: Starts the game if the given player created the room or won the last round.
 - cancelGame: Cancels the game of a room if the player can start the game.
 - playCard: Determines if a player can play a card (or draw) and updates the player's hand accordingly.
 - requestTurn: Responds to the client with whose turn it is.
 - waitTurn: Holds a connection open for a client waiting for the current player to play their turn. Sends game state data when they finish.
 - addRule: Allows the client to add a rule to the room if they won the round.

While most of these are fairly self explanatory, the real ✨magic✨ happens in the `addRule` action. When a player wins a round, the can send the server a new rule they want to add to the game. The server will then hit OpenAI's API, sending the
rule request the player made. ChatGPT will interpret this rule and spit out a new `go` function to handle the logic. This is then added to a specific rules file for the given room, and play is resumed. Whenever the server receives a `playCard`
action, it references the rules in the file and uses `yaegi` to interpret the code at runtime. 

You may think that allowing a user free reign to prompt ChatGPT to generate code that runs on your machine *hugely* irresponsible and a terrible idea. 
And to that I say, don't worry <sub>(for legal reasons, do worry)</sub>! I nicely asked it to write secure code in the system prompt 😉.

### client.go
`client.go` is the interface the user uses to interact with the server and play the game. The general flow of the game is as follows:
1. Enter an IP or domain to connect to the server (validated by the `healthCheck` action).
2. On the main menu, choose to either create a new room with the desired settings (public vs. private, number of rounds, and starting hand size) or join a created room.
3. Play the game!
    * Players take turns playing a card off the top card of the discard pile. They can play a card in their hand by typing it into the input prompt, or by choosing to draw. If the player breaks a rule, a card will automatically be drawn and they have to play again.
    * When a player wins a round, they are prompted to add a new rule to the game.
    * The player who wins the most rounds wins the game!
  
The server and client have been designed so that multiple rooms can be playing at the same time. Will enough people play this game (let alone know it exists) to warrant this fact? Probably not, but it was fun to implement anyway.

## Usage
### Prerequisites
The server requires an OpenAI API token to be able to generate new rules. If you don't have one already, you can create a new key [here](https://platform.openai.com/api-keys). 
Do note this will incur a small fee each time a player adds a rule due to API costs.

Then, export it as an environment variable called `OPENAI_TOKEN`:

```export OPENAI_TOKEN=<your API key>```

You will also need `go 1.25.5` if you plan to run it on your machine directly. Alternatively, a `Dockerfile` has been provided for building/running the code in a container 
(which is **highly recommended** due to the aforementioned... "security flexible features"). This will require having `docker` and `make` installed.

Finally, clone the repository with `git clone https://github.com/sheeple5/mao.git`.

### Running the Server
There are a multitude of options for running the server. In order of most recommended to least, perform whatever option you choose in the root of the repository.

#### Running in a Docker Container
A `docker-compose.yml` file has been provided to start the container with Docker Compose. To run the server, make sure you've set your 
`OPENAI_TOKEN` variable and simply use:

`docker compose up -d`

To stop the container, use:

`docker compose down`

Or, manual image building and execution, a `Dockerfile` and `Makefile` have been provided as well to make the image and run the server via `docker run`.
First, build the image with:

`make build`

Then, assuming you have set your `OPENAI_TOKEN` variable, start the server in a container:

`make run_server`

To stop the container, use:

`make stop`

In either case, the container should have an exposed `9090` port that can be reached through `localhost`.

#### Build server.go
You can build `server.go` into a binary first before executing. To build directly, you can run:

`go build -o mao_server server/server.go`

Alternatively, you can use the Docker image to build the binary. To run using Docker Compose, run:

`make build`

`make build_server`

Then, to run the server in either instance, execute the binary. This may require using `chmod` to give it executable permissions:

`./mao_server`

#### Running server.go directly
To run `server.go` directly, you can execute the following command:

`go run server/server.go`.

### Running the Client
Like the server, there are a multitude of similar options, in order of most recommended to least.

#### Running in a Docker Container
A `Dockerfile` and `Makefile` have been provided to make running the client in a container very simple. First, build the image with:

`make build`

Then, run the client in the container:

`make run_client`

Note that the server you are trying to connect to must be reachable by your container. A Mao server also running in a container with `make run_server` does not automatically bridge like with `docker-compose`, so you'll
have to find the server container's IP directly.

#### Build client.go
You can build `client.go` into a binary first before executing. To build directly, you can run:

`go build -o mao_client client/client.go`

Alternatively, you can use the Docker image to build the binary:

`make build`

`make build_client`

Then, to run the server in either instance, execute the binary. This may require using `chmod` to give it executable permissions:

`./mao_client`

## Closing Remarks
This was a fun project for me to practice writing in Go and try out some socket programming. It also allowed me to experiment with using AI as a feature element rather than my annoying assistant.
I personally find using AI to code very boring and stunts overall learning/growth. I learned to program because I *like* programming, not asking someone else to do it for me!

However, I do think there is a very interesting space for AI as an organic, dynamic element at runtime. Imagine trying to implement this same concept before AI! The best you'd get is either
having a preset selection of rules that players can stack on top of each other, or have the target audience be for developers and let them write code for their rules (which is equally as dangerous, mind you). Using AI
strikes a fun balance, where non-technical users can enter regular language and have the program naturally introduce it through flexibly generated code. This kind of thing I find very exciting and adds a more organic
feeling to a program that couldn't quite be achieved so smoothly before.

If you got to the end of this README... GG 🙂.







