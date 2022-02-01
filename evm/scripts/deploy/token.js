async function main() {
    const Token = await ethers.getContractFactory("Token");
    const totalSupply = ethers.utils.parseEther("10").toString()
    const token = await Token.deploy(totalSupply);

    const symbol = await token.symbol();
    console.log(symbol, "token deployed to:", token.address);
}

main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error(error);
      process.exit(1);
    });
