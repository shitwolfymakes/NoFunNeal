### FIRST WORKING QUERY
```grapqhql
{
  queryCombo(func: type(Combo)) {
    A {name}
    B {name}
    ComboResult {name}
  }
}
```

### BETTER FIRST WORKING QUERY
get the number of nodes with a combination of input A and B
```grapqhql
{
  queryCombo(func: type(Combo)) @filter(((eq(A, "Water") AND eq(B, "Fire")) OR (eq(A, "Fire") AND eq(B, "Water")))){
    uid
  }
}
```

### QUERYING ALL RESULT NODES
```grapqhql
{
  nodes(func: has(dgraph.type)) @filter(eq(dgraph.type, "Result")) {
    uid
    name
    encodedName
    emoji
    isNew
  }
}
```

### QUERY COMBO NODES OF A SPECIFIC NAME
```grapqhql
{
  queryCombo(func: type(Combo)) @filter(eq(ComboResult, "Steam")){
    A
    B
    ComboResult
  }
}
```
