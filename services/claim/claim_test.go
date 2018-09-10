package claimsrv

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/iden3/go-iden3/cmd/id/config"
	common3 "github.com/iden3/go-iden3/common"
	"github.com/iden3/go-iden3/core"
	"github.com/iden3/go-iden3/merkletree"
	"github.com/iden3/go-iden3/services/web3"
	"github.com/iden3/go-iden3/utils"
	"github.com/stretchr/testify/assert"
)

const (
	testPrivKHex = "da7079f082a1ced80c5dee3bf00752fd67f75321a637e5d5073ce1489af062d8"
	gethURL      = "https://ropsten.infura.io/TFnR8BWJlqZOKxHHZNcs"
)

var mt *merkletree.MerkleTree

func newTestingMerkle(numLevels int) (*merkletree.MerkleTree, error) {
	dir, err := ioutil.TempDir("", "db")
	if err != nil {
		return &merkletree.MerkleTree{}, err
	}
	sto, err := merkletree.NewLevelDbStorage(dir)
	if err != nil {
		return &merkletree.MerkleTree{}, err
	}

	mt, err := merkletree.New(sto, numLevels)
	return mt, err
}
func initializeEnvironment() error {
	// initialize
	config.MustRead("../../cmd/relay", "config")
	// MerkleTree leveldb
	var err error
	mt, err = newTestingMerkle(140)
	if err != nil {
		return err
	}

	// Ethereum
	err = web3srv.Open(gethURL, testPrivKHex)
	if err != nil {
		return err
	}
	return nil
}

func TestGetNextVersion(t *testing.T) {
	initializeEnvironment()

	claim := core.NewClaimDefault("c1", "c1", []byte("c1"))

	version, err := GetNextVersion(mt, claim.Hi())
	assert.Nil(t, err)
	assert.Equal(t, uint32(0), version)

	claim.BaseIndex.Version = version
	mt.Add(claim)
	version, err = GetNextVersion(mt, claim.Hi())
	assert.Nil(t, err)
	assert.Equal(t, uint32(1), version)

	claim.BaseIndex.Version = version
	mt.Add(claim)
	version, err = GetNextVersion(mt, claim.Hi())
	assert.Nil(t, err)
	assert.Equal(t, uint32(0x1000001), version)

	claim.BaseIndex.Version = version
	mt.Add(claim)
	version, err = GetNextVersion(mt, claim.Hi())
	assert.Nil(t, err)
	assert.Equal(t, uint32(0x1000002), version)

	claim.BaseIndex.Version = version
	mt.Add(claim)
	version, err = GetNextVersion(mt, claim.Hi())
	assert.Nil(t, err)
	assert.Equal(t, uint32(0x2000002), version)
}
func TestAssignNameClaim(t *testing.T) {
	initializeEnvironment()
	testPrivK, err := crypto.HexToECDSA(testPrivKHex)
	assert.Nil(t, err)

	mt.Add(core.NewClaimDefault("c1", "c1", []byte("c1")))
	mt.Add(core.NewClaimDefault("c2", "c2", []byte("c2")))
	mt.Add(core.NewClaimDefault("c3", "c3", []byte("c3")))

	namespaceHash := merkletree.HashBytes([]byte(config.C.Namespace))
	nameHash := merkletree.HashBytes([]byte("johndoe"))
	domainHash := merkletree.HashBytes([]byte(config.C.Domain))
	ethID := crypto.PubkeyToAddress(testPrivK.PublicKey)
	assignNameClaim := core.NewAssignNameClaim(namespaceHash, nameHash, domainHash, ethID)
	// signature, err := utils.Sign(assignNameClaim.Ht(), testPrivK)
	// assert.Nil(t, err)
	// signatureHex := common3.BytesToHex(signature)
	// assignNameClaimMsg := AssignNameClaimMsg{
	// 	assignNameClaim,
	// 	signatureHex,
	// }
	privK, err := crypto.HexToECDSA(config.C.Server.PrivK)
	assert.Nil(t, err)
	root, mp, _, err := AddAssignNameClaim(mt, assignNameClaim, config.C.ContractsAddress.Identities, privK)
	assert.Nil(t, err)
	mtRoot := mt.Root()
	if !bytes.Equal(root[:], mtRoot[:]) {
		t.Errorf("root != mt.Root")
	}
	expectedRootHex := "0x05175b7c17ea772423da35f9ccd0bb0017355a135e60ba28e541f26e1185b31e"
	if mt.Root().Hex() != expectedRootHex {
		t.Errorf("mt.Root: " + mt.Root().Hex() + " , expected root: " + expectedRootHex)
	}
	expectedMPHex := "0x000000000000000000000000000000000000000000000000000000000000000311a689079d0478b829d23ae5fb3e65ab15ad1abc364eea2965abf1c324e72e817370e48c8a338794dd181314bbd080e4263a802803686bcc2c2d3f554e3a50de"
	if common3.BytesToHex(mp) != expectedMPHex {
		t.Errorf("mp: " + common3.BytesToHex(mp) + " , expected mp: " + expectedMPHex)
	}
}

func TestResolvAssignNameClaim(t *testing.T) {
	namespaceHash := merkletree.HashBytes([]byte(config.C.Namespace))
	nameHash := merkletree.HashBytes([]byte("johndoe"))
	domainHash := merkletree.HashBytes([]byte(config.C.Domain))
	testPrivK, err := crypto.HexToECDSA(testPrivKHex)
	ethID := crypto.PubkeyToAddress(testPrivK.PublicKey)
	originalAssignNameClaim := core.NewAssignNameClaim(namespaceHash, nameHash, domainHash, ethID)
	assignNameClaim, err := ResolvAssignNameClaim(mt, "johndoe@iden3.io", config.C.Namespace)
	if err != nil {
		t.Errorf(err.Error())
	}
	if !bytes.Equal(assignNameClaim.Bytes(), originalAssignNameClaim.Bytes()) {
		t.Errorf("resolved AssignNameClaim != original AssignNameClaim")
	}
}

func TestNewAuthorizeKSignClaim(t *testing.T) {
	testPrivK, err := crypto.HexToECDSA(testPrivKHex)
	if err != nil {
		t.Errorf(err.Error())
	}
	testAddr := crypto.PubkeyToAddress(testPrivK.PublicKey)

	authorizeKSignClaim := core.NewAuthorizeKSignClaim("iden3.io", testAddr, "app1", "appauthz", 1535208350, 1535208350)
	signature, err := utils.Sign(authorizeKSignClaim.Ht(), testPrivK)
	assert.Nil(t, err)
	signatureHex := common3.BytesToHex(signature)
	authorizeKSignClaimMsg := AuthorizeKSignClaimMsg{
		authorizeKSignClaim,
		signatureHex,
	}
	claimProof, idRootProof, err := AddAuthorizeKSignClaim(mt, testAddr, authorizeKSignClaimMsg, config.C.ContractsAddress.Identities)
	assert.Nil(t, err)
	assert.Equal(t, "0x771e1ef9fab9bdf7f55ba7c24112b9c4b9d7e55cd94f57efd0fd4ef174565b66", mt.Root().Hex())

	// check userIDRoot
	stoUserID := mt.Storage().WithPrefix(testAddr.Bytes())
	userMT, err := merkletree.New(stoUserID, 140)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, "0x8112699ee0bb1a6307dce979a72d77549fdcf1d59648b424c5d65d5080d4b3fa", userMT.Root().Hex())

	expectedClaimProof := "0x0000000000000000000000000000000000000000000000000000000000000000"
	assert.Equal(t, expectedClaimProof, common3.BytesToHex(claimProof))
	expectedIdRootProof := "0x000000000000000000000000000000000000000000000000000000000000000730c5c5fe05516470d1963cde3ecc1b93b73b2b4d09e37a4151685d6af5260705d827465cbe023bbcfa073720ce38ab510064b1743310cca89b00fb807ca3b37e7370e48c8a338794dd181314bbd080e4263a802803686bcc2c2d3f554e3a50de"
	assert.Equal(t, expectedIdRootProof, common3.BytesToHex(idRootProof))

}

func TestMultipleAuthorizeKSignClaim(t *testing.T) {
	privKHex := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	testPrivK, err := crypto.HexToECDSA(privKHex)
	assert.Nil(t, err)
	testAddr := crypto.PubkeyToAddress(testPrivK.PublicKey)

	authorizeKSignClaim := core.NewAuthorizeKSignClaim("iden3.io", testAddr, "app1", "appauthz", 1535208355, 1535208355)
	signature, err := utils.Sign(authorizeKSignClaim.Ht(), testPrivK)
	assert.Nil(t, err)
	signatureHex := common3.BytesToHex(signature)
	authorizeKSignClaimMsg := AuthorizeKSignClaimMsg{
		authorizeKSignClaim,
		signatureHex,
	}
	claimProof, idRootProof, err := AddAuthorizeKSignClaim(mt, testAddr, authorizeKSignClaimMsg, config.C.ContractsAddress.Identities)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, "0xab8da27ef1d44f3853242f095892280390d60932f2dfdd6a9988a67f6cec35ec", mt.Root().Hex())

	stoUserID := mt.Storage().WithPrefix(testAddr.Bytes())
	userMT, err := merkletree.New(stoUserID, 140)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, "0xbdb2b31ecb9c674995f29a9bdb74065172a85e0e135c56274f8e17137451c684", userMT.Root().Hex())

	expectedClaimProof := "0x0000000000000000000000000000000000000000000000000000000000000000"
	assert.Equal(t, expectedClaimProof, common3.BytesToHex(claimProof))
	expectedIdRootProof := "0x0000000000000000000000000000000000000000000000000000000000000007585169e90e5f14f720529326b75be5fe9c4fbe0e78874c8db3c2c0fe879c87062fd3493fd39f4bd7a626383d2617bf4ead5e47941cdbe4e941edcb0bb8626b357370e48c8a338794dd181314bbd080e4263a802803686bcc2c2d3f554e3a50de"
	assert.Equal(t, expectedIdRootProof, common3.BytesToHex(idRootProof))

	privKHex2 := "a247c1a3ab5c894d68575fad9f9a519895732ba7b8b0c22afce255338ae8c345"
	testPrivK2, err := crypto.HexToECDSA(privKHex2)
	assert.Nil(t, err)
	testAddr2 := crypto.PubkeyToAddress(testPrivK2.PublicKey)
	authorizeKSignClaim2 := core.NewAuthorizeKSignClaim("iden3.io", testAddr2, "app1", "appauthz", 1535208355, 1535208355)
	signature2, err := utils.Sign(authorizeKSignClaim2.Ht(), testPrivK2)
	assert.Nil(t, err)
	signatureHex2 := common3.BytesToHex(signature2)
	authorizeKSignClaimMsg2 := AuthorizeKSignClaimMsg{
		authorizeKSignClaim2,
		signatureHex2,
	}
	claimProof2, idRootProof2, err := AddAuthorizeKSignClaim(mt, testAddr2, authorizeKSignClaimMsg2, config.C.ContractsAddress.Identities)
	assert.Nil(t, err)

	assert.Equal(t, "0xf6c57457fd9ebcd6c21acd511a41303f63e59e74c7c47d98fd0813a9bf39b392", mt.Root().Hex())

	stoUserID2 := mt.Storage().WithPrefix(testAddr2.Bytes())
	userMT2, err := merkletree.New(stoUserID2, 140)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, "0xfea5cdf67c17737bf9b148a6dc26449c1672b59d37116b916253f0abce72f160", userMT2.Root().Hex())
	expectedClaimProof = "0x0000000000000000000000000000000000000000000000000000000000000000"
	assert.Equal(t, expectedClaimProof, common3.BytesToHex(claimProof2))
	expectedIdRootProof = "0x000000000000000000000000000000000000000000000000000000000000001713bc31bd2a88624073b508ade2ce7e8a2207c53b12f0dbdfc4547362d6376e1312610bb2a7c84995083296c0b3eada2d57184d2b4f02adb907a649d7748c614ad25b5563e50227d3c4ff6b9161f5381a292a998ae7d53ec74960ece6a04f5fb07370e48c8a338794dd181314bbd080e4263a802803686bcc2c2d3f554e3a50de"
	assert.Equal(t, expectedIdRootProof, common3.BytesToHex(idRootProof2))
}

func TestNewUserIDClaim(t *testing.T) {
	privKHex := "da7079f082a1ced80c5dee3bf00752fd67f75321a637e5d5073ce1489af062d8"
	testPrivK, err := crypto.HexToECDSA(privKHex)
	assert.Nil(t, err)
	testAddr := crypto.PubkeyToAddress(testPrivK.PublicKey)

	claim := core.NewClaimDefault("iden3.io_3", "default", []byte("data2"))
	signature, err := utils.Sign(claim.Ht(), testPrivK)
	assert.Nil(t, err)
	signatureHex := common3.BytesToHex(signature)
	claimValueMsg := ClaimValueMsg{
		claim,
		signatureHex,
	}
	claimProof, idRootProof, err := AddUserIDClaim(mt, "iden3.io", testAddr, claimValueMsg, config.C.ContractsAddress.Identities)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, "0x964e3bb814386a83eb85ccca2f09812bdb9582afa30fe1e454c5f4dfcb6bd70e", mt.Root().Hex())

	// check userIDRoot
	stoUserID := mt.Storage().WithPrefix(testAddr.Bytes())
	userMT, err := merkletree.New(stoUserID, 140)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, "0xcda67faf66cf1261e9653c2528883f7e7c6fa4ea9ddef2a3b669817e0b2d1bbc", userMT.Root().Hex())

	expectedClaimProof := "0x000000000000000000000000000000000000000000000000000000000000000257a42f22a7e9b3acf712f7bb8a4e684f965f8c3ee2dc0fc88129c8b624754fcd"
	assert.Equal(t, expectedClaimProof, common3.BytesToHex(claimProof))
	expectedIdRootProof := "0x0000000000000000000000000000000000000000000000000000000000000107f3e6294d5cb4ef3ff284318ddce1f539111c3310e04075420b89dac28d1003b15def58d649018d988ff4d4c7cf9cbc4ab7590d58fa06e76b28f802212e2b5083f9e894a02f51799114c844c03d5859069afb4c7287a5403c6c4fba577467bed57370e48c8a338794dd181314bbd080e4263a802803686bcc2c2d3f554e3a50de"
	assert.Equal(t, expectedIdRootProof, common3.BytesToHex(idRootProof))

}
func TestGetIDRoot(t *testing.T) {
	privKHex := "da7079f082a1ced80c5dee3bf00752fd67f75321a637e5d5073ce1489af062d8"
	testPrivK, err := crypto.HexToECDSA(privKHex)
	assert.Nil(t, err)
	testAddr := crypto.PubkeyToAddress(testPrivK.PublicKey)

	idRoot, idRootProof, err := GetIDRoot(mt, testAddr)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, "0xcda67faf66cf1261e9653c2528883f7e7c6fa4ea9ddef2a3b669817e0b2d1bbc", idRoot.Hex())
	expectedProof := "0x0000000000000000000000000000000000000000000000000000000000000007ab9ed10e59863ed65028fda65d43dc320388afd2ff6510e0677d04acf376e89f4f7c6e940a2392179ceb7120d4a3127bd7918a3c0f7bf1726958523214fc73247370e48c8a338794dd181314bbd080e4263a802803686bcc2c2d3f554e3a50de"
	assert.Equal(t, expectedProof, common3.BytesToHex(idRootProof))
}

func TestGetClaimByHiThatDontExist(t *testing.T) {
	privKHex := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	testPrivK, err := crypto.HexToECDSA(privKHex)
	assert.Nil(t, err)
	testAddr := crypto.PubkeyToAddress(testPrivK.PublicKey)

	hiHex := "0x784adb4a490b9c0521c11298f384bf847881711f1a522a40129d76e3cfc68c9a"
	hiBytes, err := common3.HexToBytes(hiHex)
	assert.Nil(t, err)
	hi := merkletree.Hash{}
	copy(hi[:], hiBytes)
	_, _, _, _, _, _, err = GetClaimByHi(mt, "namespace.io", testAddr, hi)
	assert.NotNil(t, err)
}

func TestAddClaimAndGetClaimByHi(t *testing.T) {
	privKHex := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	testPrivK, err := crypto.HexToECDSA(privKHex)
	assert.Nil(t, err)
	testAddr := crypto.PubkeyToAddress(testPrivK.PublicKey)

	claim := core.NewClaimDefault("namespace.io", "default", []byte("dataasdf"))
	signature, err := utils.Sign(claim.Ht(), testPrivK)
	assert.Nil(t, err)
	signatureHex := common3.BytesToHex(signature)
	claimValueMsg := ClaimValueMsg{
		claim,
		signatureHex,
	}
	claimProof, idRootProof, err := AddUserIDClaim(mt, "namespace.io", testAddr, claimValueMsg, config.C.ContractsAddress.Identities)
	assert.Nil(t, err)
	hi := claim.Hi()
	value, idProof, idRoot, setRootClaim, relayProof, relayRoot, err := GetClaimByHi(mt, "namespace.io", testAddr, hi)
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)
	assert.Equal(t, "0xa92591b1ee18783f95fbf358517afed09d888b9db8286c0d19e2419036941d68cfee7c08a98f4b565d124c7e4e28acc52e1bc780e3887db0a02a7d2d5bc66728000000006461746161736466", common3.BytesToHex(value.Bytes()))
	assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000002546f8feb74144a5ee688f26ee5c5c202051386d6682164b1746d7481c4c5fda0", common3.BytesToHex(idProof))
	assert.Equal(t, "0x174798396a958603a3c6b2f60b21a4735000429be4d5dded269b93ba37945898", idRoot.Hex())
	assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000017c29ecab32fdae08849361b5c7140a83442ccea6a6b94fe55b1cda5e2b52681015def58d649018d988ff4d4c7cf9cbc4ab7590d58fa06e76b28f802212e2b5083f9e894a02f51799114c844c03d5859069afb4c7287a5403c6c4fba577467bed57370e48c8a338794dd181314bbd080e4263a802803686bcc2c2d3f554e3a50de", common3.BytesToHex(relayProof))
	assert.Equal(t, "0x23a44df999057bf245c43f196948bbbd7d4282dbb4d6027a30a14fcd4798aa2e", relayRoot.Hex())

	assert.Equal(t, claimProof, idProof)
	assert.Equal(t, idRootProof, relayProof)
	verified := merkletree.CheckProof(idRoot, idProof, value, 140)
	assert.True(t, verified)
	assert.Equal(t, mt.Root().Bytes(), relayRoot.Bytes())
	verified = merkletree.CheckProof(relayRoot, relayProof, setRootClaim, mt.NumLevels())
	assert.True(t, verified)

}