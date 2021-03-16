## lightChain

Several days ago, I learned the blockchain techniques from the online lectures 
[Blockchain: techniques and applications](http://zhenxiao.com/blockchain/), taught by 
[Zhen Xiao](http://zhenxiao.com) from PKU. I was deeply attracted by the beauty of the 
underlying principles of the blockchain. To test the effect of learning, I build a light-weight 
blockchain (named lightChain) starting from scratch in this repo. When writing lightChain, I refer 
to several open-source projects such as the famous [go-ethereum](https://github.com/ethereum/go-ethereum), 
the [bitcoin/bitcoin](https://github.com/bitcoin/bitcoin) , and several guidance-oriented blogs 
(see references).


In lightChain, we basically use PoW for the mining of new blocks. I may add the implementation of 
PoS and some other variants in the future. It depends. :-)

Steps | Contents | Progress
--- | --- | ---
1 | Add the blockchain cores | <ui><li>Add wallets ☑️</li><li>Implement the pseudo P2P network ☑️</li><li>Implement merkle tree ☑️</li><li>Update UTXO data structure ☑️</li></ui>
2 | Update the cli | <ui><li>Print lightChain ☑️</li><li>Print pointed transaction ☑️</li><li>List addresses ☑️</li><li>Reindex UTXO</li></ui>
3 | Publish the executable and docker image | <ui><li>Publish the executable ☑️</li><li>Publish the docker image</li></ui>


#### References

* [Blockchain: techniques and applications](http://zhenxiao.com/blockchain/)
* [ethereum/go-ethereum](https://github.com/ethereum/go-ethereum)
* [bitcoin/bitcoin](https://github.com/bitcoin/bitcoin)  
* [Building Blockchain in Go](https://jeiwan.net/posts/building-blockchain-in-go-part-1/)