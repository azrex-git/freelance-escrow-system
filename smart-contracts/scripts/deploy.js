import hre from "hardhat";

async function main() {
  const Escrow = await hre.ethers.getContractFactory("FreelanceEscrow");
  const escrow = await Escrow.deploy();

  await escrow.waitForDeployment();

  console.log("FreelanceEscrow deployed to:", await escrow.getAddress());
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
