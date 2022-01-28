async function main() {
    const Token = await ethers.getContractFactory("Token");
    const token = await Token.deploy(1*10**8);

    const symbol = await token.symbol();
    console.log(symbol, "token deployed to:", token.address);
}

main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error(error);
      process.exit(1);
    });
