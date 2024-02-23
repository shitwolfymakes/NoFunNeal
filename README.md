# Just how "infinite" is Infinite Craft?
### What is Infinite Craft?
[Infinite Craft](https://neal.fun/infinite-craft/) is combiner game in the same vein as Doodle God, with the twist that responses are generated by Llama 2. This makes the combinations theoretically infinite, but potentially not if the LLM starts to hallucinate.

### What does it look like under the hood?
The API is called with a URL constructed from the two types being combined:
```
https://neal.fun/api/infinite-craft/pair?first=Fire&second=Water
```
And returns a JSON block with this structure:
```json
{
  "result":"Steam",
  "emoji":"💨",
  "isNew":false
}
```

### How can we store the response?  
A graph database is ideal for this. I went with the DGraph database because it had a docker compose that worked the moment I ran it  

### How many responses can we send responsibly? (Smant/PointCrow Infinite-Craft stream for numbers)  
Rate limit script only reaches a 100 requests in roughly 30 secs, so DDoS isn't something that needs to be worried about with a single runner.

Full testing with a single agent hit a rate limit of roughly 1 request/second

### Api hit failure (I think he noticed me)
A couple of days after starting curl testing on the api, it suddenly started returning 403 errors. Turns out I needed to include User Agent String!

### What size is the problem space?
Given the combination is of any 2 items, and we can't rule out that order doesn't matter, the potential total number of combinations are:
```
∑,M,k=0; k!/((M−k)!M!) // the formula for total unique combinations
```

### Algos for exhaustive search of M-to-M combinations
I present: `Bogosearch`, like Bogosort, but for searching. Maximally cursed!
1. Select two random types
2. Check if this combination is already stored
3. If not already stored, reach out to the api to get the combination

### Publishing the data
TBD

## Stuff I learned
- Golang!
- Graph database basics, DGraph, GraphQL
- Document database basics, MongoDB
- API analysis and communication
- Improved prompting skills for/pair programming with ChatGPT, even 3.5 is pretty great if you know what you're asking for
- Secrets management and handling creds programmatically (Biggest weakness is hardcoding filenames)

## Headaches encountered
- DGraph doesn't appear to have a way to spin up the db using a custom schema (wtf?)
- Containerizing the agent itself was a 5-hour rabbithole of intractible networking issues. Skipped for time. Ultimately unnecessary, as the rate limit is super low.
