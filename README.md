# dgtrello
A Discord bot made with [discordgo](https://github.com/bwmarrin/discordgo) in order to notify events from Trello boards to Discord server.

## Usage
Create your Trello Power-Up [here](https://trello.com/power-ups/admin), then generate the Trello API key and token of the Power-Up.
Setup your `config.json` file as the example below:
```json
{
  "adminRoles": [
    "<Role allowed to use the `trello` command>"
  ],
  "channels": [
    {
      "channelId": "<Your channel Id>",
      "boardId": "<Trello board id to listen>",
      "enabledEvents": [
        "createCard",
        "copyCard",
        "commentCard",
        "deleteCard",
        "updateCard"
      ],
      "lastActionId": "6408ceabbddcacfe1ed9ade9"
    }
  ],
  "cmdPrefix": "!",
  "discordToken": "<Your discord bot token>",
  "members": {
    "<trello username>": "<discord userid>"
  },
  "pollInterval": 1000,
  "trelloApiKey": "<Your trello api key>",
  "trelloToken": "<Your trello auth token>"
}

```
Run the bot executable to start logging events on the configured channels
```bash
dgtrello --config=config.json
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.
