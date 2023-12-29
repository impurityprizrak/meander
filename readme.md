# Meander

## General Purposes
This is an experimental program written in Go that I've designed to learn more about the language. It's a blockchain-like project that provides a source code to startup node servers and interact with other peers in the network, enabling the creation of transactions and the blocks mining.

The project structure is as follows:

- The node acts as the server to which requests can be sent.
- Clients can register themselves with a given node.
- Clients can make transactions to other clients at any node in the network.
- Transactions can be mined by miners to create blocks that are then added to the chain.

All nodes have a chain, and the process of mining and adding blocks follows these steps:

- Each client has a pair of keys used to sign its transactions.
- Transactions are collapsed into a block, along with the hash of the last block and the nonce.
- Miners work to generate the hash of the new block with a difficulty of 5, expecting a hash with at least five zeros at the left.
- Nodes must find a consensus on the correct version of the chain.

A node consists of a gRPC server with ElasticSearch as the database to store clients, transactions, and blocks. Each node has its own database and shares transactions and chain blocks with other nodes through gRPC calls.

---
## Up the server

No informations about running the program yet. There will be possible to run the Meander using docker compose.
