require("@nomiclabs/hardhat-waffle");

// You need to export an object to set up your config
// Go to https://hardhat.org/config/ to learn more

/**
 * @type import('hardhat/config').HardhatUserConfig
 */
module.exports = {
  solidity: "0.8.4",
  defaultNetwork: "local",
  networks: {
    local: {
      url: "http://localhost:8545/",
      accounts: ["C93F165DF8EC9D318A464CA9304E96D627674DC7CD745B97786BB696480F13B3"]
    }
  },
};
