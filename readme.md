# Dusk Wallet

    The transaction structure is still being modified. 
    
    - Inputs, outputs, ring sig and bulletproof are likely to change.

## Cryptographic Primitives and Assumptions

- All scalars are assumed to be within the subgroup of the curve used. This assumption is justified as we are using ristretto, where we can think of the co-factor as being 1.

- Assume `G` to be the standard basepoint and `H` to be some other generator such that we do not know the discrete log, with respects to `G`.

- Pedersen Commitment is a point such that C = aH + bG

- HashToGroup; given arbitrary data, map it to some element in that group.

## Keys 

*Private* Spend Key:

    - seed = Rand()
    - PrivateSpendKey = HashToScalar(seed)

*Public* Spend Key:

    - PublicSpendKey = PrivateSpendKey * G


*Private* View Key:

    - PrivateViewKey = HashToScalar(PrivateSpendKey.Bytes())

*Public* View Key:

    - PublicViewKey = PrivateViewKey * G


### Stealth Address (One time public key)

    Alice wants to send money to Bob.

    Alice is given Bob's Public Spend(BPs) and Public View keys(BPw).

    Alice generates a one time public key (P) for Bob:

        - r = Rand()
        - P = (HashToScalar(r * PubView || Index) + privSpend) * G
        - Note to spend, one must know: (HashToScalar(r * PubView || Index) + privSpend)

        - The index is a number that is transaction specific. It's usage is expanded upon in the transactions section.

### Public Address

    Alice generates her own public address by :

    Sum = Hash(netIdentifier + pubViewKey + PubSpendKey)
    checksum = Sum[:4]

    pubAddr = Base58Encode(netIdentifier + pubViewKey + PubSpendKey + checksum)

## Stealth Transaction

## Components of a stealth transaction:

    - Transaction PubKey
    - Outputs
    - Inputs
    - Fee

### Transaction Pubkey

    To generate a transaction public key `R`:

    - r = rand()
    - R = r *G

    Note: In a transaction, this is the same r used to generate the stealth address.

### Outputs

Outputs consist of a destination key, encrypted amount, encrypted mask, commitment and a rangeproof.

    - The *destination key* is synonymous with the one-time public key. However, since a one-time public key can only be used once. If Alice sends multiple outputs to one stealth address, and all of the outputs are not spent at the same time. The remaining unspent outputs will not be usable. To solve this, each output asscosciates an index with the corresponding output. This way, each output will generate a different one-time public key, whereby the sender can recover the private key by knowing what index the output is at.

    - mask = rand()
    - *Commitment*; P = amount * G + mask * H 

    - *EncryptedAmount* = amount + H(H(H(r*PubViewKey || index)))
    - *EncryptedMask* = mask + H(H(r*PubViewKey || index))
    - If two amounts and masks are similar, the difference amount of hashes will ensure that there is no similarity in the encrypted amounts. This would only be a problem if a bad hashing function was used, or a bad rng was used.

    - *Rangeproof* = Prove(amount)

### Inputs

    Inputs consist of a TxID, Commitment, KeyImage, Decoys and RingSig.

    - *TxID* of the previous transaction that this input was an output.

    - *Commitment* assosciated commitment when input was an output.

    - *KeyImage* image of private key used in ring sig

    - *RingSig* ring signature used to hide  

    - *Decoys* decoy transactions


## Contributions 

The above scheme has been modified from the monero codebase.