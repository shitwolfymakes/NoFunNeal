### EDIT SCHEMA
In Schema -> Bulk Edit -> replace with the following and press "Apply Schema"

```
<A>: string @index(fulltext, term) .
<B>: string @index(fulltext, term) .
<ComboResult>: string @index(fulltext, term) .
<name>: string @index(fulltext, term) .
<encodedName>: string @index(fulltext, term) .
<emoji>: string .
<isNew>: bool .
type <Combo> {
	A
	B
	ComboResult
}
type <Result> {
	name
	encodedName
	emoji
	isNew
}
```

### Load data
In Console -> Mutate -> Paste the following and press "Run"
```
{
  set {
    _:water <dgraph.type> "Result" .
    _:water <name> "Water" .
    _:water <encodedName> "Water" .
    _:water <emoji> "💧" .
    _:water <isNew> "false" .

    _:fire <dgraph.type> "Result" .
    _:fire <name> "Fire" .
    _:fire <encodedName> "Fire" .
    _:fire <emoji> "🔥" .
    _:fire <isNew> "false" .

  	_:wind <dgraph.type> "Result" .
    _:wind <name> "Wind" .
    _:wind <encodedName> "Wind" .
    _:wind <emoji> "🌬️" .
    _:wind <isNew> "false" .

  	_:earth <dgraph.type> "Result" .
    _:earth <name> "Earth" .
    _:earth <encodedName> "Earth" .
    _:earth <emoji> "🌍" .
    _:earth <isNew> "false" .

  	_:steam <dgraph.type> "Result" .
    _:steam <name> "Steam" .
    _:steam <encodedName> "Steam" .
    _:steam <emoji> "💨" .
    _:steam <isNew> "false" .

  	_:combo_steam <dgraph.type> "Combo" .
    _:combo_steam <A> "Water" .
    _:combo_steam <B> "Fire" .
    _:combo_steam <ComboResult> "Steam" .
  }
}
```
