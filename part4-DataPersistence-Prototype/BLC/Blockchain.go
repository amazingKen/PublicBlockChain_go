/**
@author: chaors

@file:   Blockchain.go

@time:   2018/06/21 22:40

@desc:   区块链基础结构
*/

package BLC

import (
	"github.com/boltdb/bolt"
	"log"
	"fmt"
	"math/big"
	"time"
	"os"
)

//相关数据库属性
const dbName = "chaorsBlockchain.db"
const blockTableName = "chaorsBlocks"
const newestBlockKey = "chNewestBlockKey"

type Blockchain struct {
	//最新区块的Hash
	Tip []byte
	//存储区块的数据库
	DB *bolt.DB
}

//1.创建带有创世区块的区块链
func CreateBlockchainWithGensisBlock() *Blockchain {

	var blockchain *Blockchain

	//判断数据库是否存在
	if IsDBExists(dbName) {

		db, err := bolt.Open(dbName, 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		err = db.View(func(tx *bolt.Tx) error {

			b := tx.Bucket([]byte(blockTableName))
			if b != nil {

				hash := b.Get([]byte(newestBlockKey))
				blockchain = &Blockchain{hash, db}
				//fmt.Printf("%x", hash)
			}

			return nil
		})
		if err != nil {

			log.Panic(err)
		}

		return blockchain
	}

	//创建并打开数据库
	db, err := bolt.Open(dbName, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(blockTableName))
		//blockTableName不存在再去创建表
		if b == nil {

			b, err = tx.CreateBucket([]byte(blockTableName))
			if err != nil {

				log.Panic(err)
			}
		}

		if b != nil {

			//创世区块
			gensisBlock := CreateGenesisBlock("Gensis Block...")
			//存入数据库
			err := b.Put(gensisBlock.Hash, gensisBlock.Serialize())
			if err != nil {
				log.Panic(err)
			}

			//存储最新区块hash
			err = b.Put([]byte(newestBlockKey), gensisBlock.Hash)
			if err != nil {
				log.Panic(err)
			}

			blockchain = &Blockchain{gensisBlock.Hash, db}
		}

		return nil
	})
	//更新数据库失败
	if err != nil {
		log.Fatal(err)
	}

	return blockchain
}

//2.新增一个区块到区块链
func (blc *Blockchain) AddBlockToBlockchain(data string) {

	err := blc.DB.Update(func(tx *bolt.Tx) error {

		//1.取表
		b := tx.Bucket([]byte(blockTableName))
		if b != nil {

			//2.height,prevHash都可以从数据库中取到 当前最新区块即添加后的上一个区块
			blockBytes := b.Get(blc.Tip)
			block := DeSerializeBlock(blockBytes)

			//3.创建新区快
			newBlock := NewBlock(data, block.Height+1, block.Hash)
			//4.区块序列化入库
			err := b.Put(newBlock.Hash, newBlock.Serialize())
			if err != nil {
				log.Fatal(err)
			}
			//5.更新数据库里最新区块
			err = b.Put([]byte(newestBlockKey), newBlock.Hash)
			if err != nil {
				log.Fatal(err)
			}
			//6.更新区块链最新区块
			blc.Tip = newBlock.Hash
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

//3.遍历输出所有区块信息  --> 以后一般使用优化后的迭代器方法(见3.X)
func (blc *Blockchain) Printchain1() {

	var block *Block
	//当前遍历的区块hash
	var curHash []byte = blc.Tip
	for {

		err := blc.DB.View(func(tx *bolt.Tx) error {

			b := tx.Bucket([]byte(blockTableName))
			if b != nil {
				blockBytes := b.Get(curHash)
				block = DeSerializeBlock(blockBytes)

				/**时间戳格式化 Format里的年份必须是固定的！！！
				这个好像是go诞生的时间
				time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05")
				"2006-01-02 15:04:05"格式固定，改变其他也可能会出错
				*/
				fmt.Printf("\n#####\nHeight:%d\nPrevHash:%x\nHash:%x\nData:%s\nTime:%s\nNonce:%d\n#####\n",
					block.Height, block.PrevBlockHash, block.Hash, block.Data, time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"), block.Nonce)
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)
		//遍历到创世区块，跳出循环  创世区块哈希为0
		if big.NewInt(0).Cmp(&hashInt) == 0 {

			break
		}
		curHash = block.PrevBlockHash
	}
}

//3.X 优化区块链遍历方法
func (blc *Blockchain) Printchain() {
	//迭代器
	blcIterator := blc.Iterator()
	for {

		block := blcIterator.Next()
		fmt.Printf("\n#####\nHeight:%d\nPrevHash:%x\nHash:%x\nData:%s\nTime:%s\nNonce:%d\n#####\n",
			block.Height, block.PrevBlockHash, block.Hash, block.Data, time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"),block.Nonce)

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)

		if big.NewInt(0).Cmp(&hashInt) == 0 {

			break
		}
	}
}


//生成当前区块链迭代器的方法
func (blc *Blockchain) Iterator() *BlockchainIterator {

	return &BlockchainIterator{blc.Tip, blc.DB}
}

//判断数据库是否存在
func IsDBExists(dbName string) bool {

	//if _, err := os.Stat(dbName); os.IsNotExist(err) {
	//
	//	return false
	//}

	_, err := os.Stat(dbName)
	if err == nil {

		return true
	}
	if os.IsNotExist(err) {

		return false
	}

	return true
}