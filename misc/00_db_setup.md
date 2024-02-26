### EDIT SCHEMA
In Schema -> Bulk Edit -> replace with the following and press "Apply Schema"

```
<A>: string @index(exact, fulltext) .
<B>: string @index(exact, fulltext) .
<ComboResult>: string @index(exact, fulltext) .
<result>: string @index(exact, fulltext) .
<encodedName>: string @index(exact, fulltext) .
<emoji>: string .
<isNew>: bool .
type <Combo> {
	A
	B
	ComboResult
}
type <Result> {
	result
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
    _:water <result> "Water" .
    _:water <encodedName> "Water" .
    _:water <emoji> "💧" .
    _:water <isNew> "false" .

    _:fire <dgraph.type> "Result" .
    _:fire <result> "Fire" .
    _:fire <encodedName> "Fire" .
    _:fire <emoji> "🔥" .
    _:fire <isNew> "false" .

  	_:wind <dgraph.type> "Result" .
    _:wind <result> "Wind" .
    _:wind <encodedName> "Wind" .
    _:wind <emoji> "🌬️" .
    _:wind <isNew> "false" .

  	_:earth <dgraph.type> "Result" .
    _:earth <result> "Earth" .
    _:earth <encodedName> "Earth" .
    _:earth <emoji> "🌍" .
    _:earth <isNew> "false" .

  	_:steam <dgraph.type> "Result" .
    _:steam <result> "Steam" .
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

### QUERY TO CONFIRM DATA WAS LOADED
In Console -> Query -> Paste the following and press "Run"
```
{
    nodes(func: has(dgraph.type)) {
        uid
        result
        encodedName
        emoji
        isNew
    }
}
```
