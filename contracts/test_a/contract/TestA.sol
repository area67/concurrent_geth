pragma solidity >=0.5.7;

contract TestA {
    uint a;

    function increment() public {
        a += 1;
    }

    function decrement() public {
        a -= 1;
    }

    function get() public view returns (uint) {
        return a;
    }
}