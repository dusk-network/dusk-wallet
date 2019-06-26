This version uses the public keys in the output as key_offsets

This will add on average an extra 20 bytes. The reason why it is like this is due to the mlsag mixing the keys 
and then needing to have the offsets match order with the new mixed order. 

    - This can be mitigated by instead of shuffling, we have the mlsag order the signers by public keys in lexicographical order. This would introduce another edge case.
    - The modified version is simpler because we can just read off of the public keys from the output and we can get the order from the mlsag signature directly.



// Block Transactions can be ordered further:


Block {

    TX {

        []TxType1{

        }
        []TxType2{

        }
        []TxType3{

        }

    }
}

Encoding:


lenTxs:=encode(len(txs))
lenTyp1 := encode(len(txtype1))

// Optimistically use a uint16 data type to count the tx types

stake, bid, timelock, standard,coinbase

coinbase will be first and will always have one per block, so no counter needed

uint16 = 2 bytes 
hay una 4 types of txs so 8 bytes per block
Is this worth the code simplicity? Will there be more tx types in the future?

For more tx types, we can upgrade block version and add, but this approach may not be great if there are going to be a lot more tx types to add

Are we saving any bytes?

We can remove the `tx type` from each tx which is a uint8 = 1 byte

If we have N transactions then we will use N bytes to specify the transaction type for each

However, if we use the updated approach, we will need to have 8 bytes in total to specify the grouped types (4 uint16s)

`Space saving = N - 8` , so we start saving when we have more than 9 transactions per block




