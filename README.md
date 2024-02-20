# Just how "infinite" is Infinite Craft?
- notes on what infinite craft is
- What does it look like under the hood?
- how can we send a get request programatically
- how can we store the response  
Graph database is perfect for this. I went with the DGraph database bc it had a docker compose that worked the moment I ran it
- how many responses can we send responsibly? (Smant/PointCrow Infinite-Craft stream for numbers)  
Rate limit script only reaches a 100 requests in roughly 30 secs, so DDoS isn't something that needs to be worried about with a single runner.
- Api hit failure (I think he noticed me)
- What big-O time is the problem space?
- algos for exhaustive search of M-to-M combinations
- publishing the data

### Stuff I learned
- Golang!
- Graph database basics, DGraph, GraphQL
- Document database basics, MongoDB
- API analysis and communication
- Improved prompting skills for/pair programming with ChatGPT, even 3.5 is pretty great if you know what you're asking for

### Headaches encountered
- DGraph doesn't appear to have a wait to spin up using a custom schema (wtf?)
- Containerizing the agent itself was a 5-hour rabbithole of intractible networking issues. Skipped for time
