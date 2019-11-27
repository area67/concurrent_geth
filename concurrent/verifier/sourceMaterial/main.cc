//sudo apt-get install libtbb-dev
//sudo apt-get install libboost-dev

//#include <iostream>
#include <stdio.h>
#include <thread>
#include <chrono>
#include <list>
#include <atomic>
//#include <queue>          // std::priority_queue
//#include <vector>         // std::vector
#include <map>
#include <unordered_map>
#include <climits>
#include <stack>          // std::stack
#include <math.h>       /* pow */
#include <boost/random.hpp>
#include <sys/time.h>

#include <tbb/concurrent_queue.h>
#include <boost/lockfree/stack.hpp>
#include "tbb/concurrent_hash_map.h"

#define NUM_THRDS 32
//#define TEST_SIZE 100
unsigned int TEST_SIZE;

//Set one parameter to 1 and the rest 0
#define LINEARIZABILITY 0
#define SEQUENTIAL_CONSISTENCY 0
#define SERIALIZABILITY 1

#define DEBUG_ 0

//Set one parameter to 1 and the rest 0
//#define TBB_QUEUE 0
//#define BOOST_STACK 0
//#define TBB_MAP 0

unsigned int TBB_QUEUE;
unsigned int BOOST_STACK;
unsigned int TBB_MAP;

struct MyHashCompare {
    static size_t hash( int x ) {
        return x;
    }
    //! True if strings are equal
    static bool equal( int x, int y ) {
        return x==y;
    }
};

typedef enum Status
{
	PRESENT,
	ABSENT
}Status; 

typedef enum Semantics
{
	FIFO,
	LIFO,
	SET,
	MAP,
	PRIORITY
}Semantics; 

typedef enum Type
{
	PRODUCER,
	CONSUMER,
	READER,
	WRITER
}Type; 

typedef struct Method
{
	int id; //atomic var 
	// int process; 
	//int item;
	int item_key;
	int item_val;
	Semantics semantics; //hardcode as set
	Type type; // subtracting adding 
    // long int invocation; 
	// long int response;
	// int quiescent_period;
	bool status;
	// int txn_id; //Transaction id, assigned according to transaction order

	Method(int _id, int _process, int _item_key, int _item_val, Semantics _semantics, Type _type, long int _invocation, long int _response, bool _status, int _txn_id) 
	{
		id = _id;
		process = _process;
		item_key = _item_key;
		item_val = _item_val;
		semantics = _semantics;
		type = _type;
		invocation = _invocation;
		response = _response;
		quiescent_period = -1;
		status = _status;
		txn_id = _txn_id;
	}
}Method;

typedef struct Item
{
	int key;
	int value;
	//int sum;
	double sum;
	
	//READS needs a separate sum, check linearizability by taking ceiling of sum + sum_reads > 0

	long int numerator;
	long int denominator;	

	double exponent; 

	Status status;
	
	std::stack<int> promote_items;

	std::list<Method> demote_methods;

	std::map<long int,Method,bool(*)(long int,long int)>::iterator producer;

	//Failed Consumer
	double sum_f;
	long int numerator_f;
	long int denominator_f;	
	double exponent_f;

	//Reader
	double sum_r;
	long int numerator_r;
	long int denominator_r;	
	double exponent_r;

	Item(int _key) 
	{
		key = _key;
		value = INT_MIN;
		sum = 0;
		numerator = 0;
		denominator = 1;
		exponent = 0;
		status = PRESENT;
		sum_f = 0;
		numerator_f = 0;
		denominator_f = 1;
		exponent_f = 0;
		sum_r = 0;
		numerator_r = 0;
		denominator_r = 1;
		exponent_r = 0;
	}

	Item(int _key, int _val) 
	{
		key = _key;
		value = _val;
		sum = 0;
		numerator = 0;
		denominator = 1;
		exponent = 0;
		status = PRESENT;
		sum_f = 0;
		numerator_f = 0;
		denominator_f = 1;
		exponent_f = 0;
		sum_r = 0;
		numerator_r = 0;
		denominator_r = 1;
		exponent_r = 0;
	}

	void add_int(long int x)
	{
		//printf("Test add function\n");
		long int add_num = x * denominator;
		numerator = numerator + add_num;
		//printf("add_num = %ld, numerator/denominator = %ld\n", add_num, numerator/denominator);
		sum = (double) numerator/denominator;
		//sum = sum + x;
	}
	void sub_int(long int x)
	{
		//printf("Test add function\n");
		long int sub_num = x * denominator;
		numerator = numerator - sub_num;
		//printf("sub_num = %ld, numerator/denominator = %ld\n", sub_num, numerator/denominator);
		sum = (double) numerator/denominator;
		//sum = sum + x;
	}
	void add_frac(long int num, long int den)
	{
#if DEBUG_
		if(den == 0)
			printf("WARNING: add_frac: den = 0\n");
		if(denominator == 0)
			printf("WARNING: add_frac: 1. denominator = 0\n");
#endif
		if(denominator%den == 0)
		{
			numerator = numerator + num * denominator/den;
		} else if (den%denominator == 0) {		
			numerator = numerator * den/denominator + num;
			denominator = den;
		} else {
			numerator = numerator * den + num * denominator;
			denominator = denominator * den;
		}
#if DEBUG_
		if(denominator == 0)
			printf("WARNING: add_frac: 2. denominator = 0\n");
#endif
		sum = (double) numerator/denominator;
	}

	void sub_frac(long int num, long int den)
	{
#if DEBUG_
		if(den == 0)
			printf("WARNING: sub_frac: den = 0\n");
		if(denominator == 0)
			printf("WARNING: sub_frac: 1. denominator = 0\n");
#endif
		if(denominator%den == 0)
		{
			numerator = numerator - num * denominator/den;
		} else if (den%denominator == 0) {		
			numerator = numerator * den/denominator - num;
			denominator = den;
		} else {
			numerator = numerator * den - num * denominator;
			denominator = denominator * den;
		}
#if DEBUG_
		if(denominator == 0)
			printf("WARNING: sub_frac: 2. denominator = 0\n");
#endif
		sum = (double) numerator/denominator;
	}
	
	void demote()
	{
		exponent = exponent + 1;
		long int den = (long int) pow (2, exponent);
		//printf("denominator = %ld\n", den);
		sub_frac(1, den);
	}

	void promote()
	{
		long int den = (long int) pow (2, exponent);
#if DEBUG_
		if(den == 0)
			printf("2 ^ %f = %ld?\n", exponent, den);
#endif
		if(exponent < 0)
			den = 1;
		//printf("denominator = %ld\n", den);
		add_frac(1, den);
		exponent = exponent - 1;
	}

	//Failed Consumer
	void add_frac_f(long int num, long int den)
	{
#if DEBUG_
		if(den == 0)
			printf("WARNING: add_frac_f: den = 0\n");
		if(denominator_f == 0)
			printf("WARNING: add_frac_f: 1. denominator_f = 0\n");
#endif
		if(denominator_f%den == 0)
		{
			numerator_f = numerator_f + num * denominator_f/den;
		} else if (den%denominator_f == 0) {		
			numerator_f = numerator_f * den/denominator_f + num;
			denominator_f = den;
		} else {
			numerator_f = numerator_f * den + num * denominator_f;
			denominator_f = denominator_f * den;
		}
#if DEBUG_
		if(denominator_f == 0)
			printf("WARNING: add_frac_f: 2. denominator_f = 0\n");
#endif
		sum_f = (double) numerator_f/denominator_f;
	}

	void sub_frac_f(long int num, long int den)
	{
#if DEBUG_
		if(den == 0)
			printf("WARNING: sub_frac_f: den = 0\n");
		if(denominator_f == 0)
			printf("WARNING: sub_frac_f: 1. denominator_f = 0\n");
#endif
		if(denominator_f%den == 0)
		{
			numerator_f = numerator_f - num * denominator_f/den;
		} else if (den%denominator_f == 0) {		
			numerator_f = numerator_f * den/denominator_f - num;
			denominator_f = den;
		} else {
			numerator_f = numerator_f * den - num * denominator_f;
			denominator_f = denominator_f * den;
		}
#if DEBUG_
		if(denominator_f == 0)
			printf("WARNING: sub_frac_f: 2. denominator_f = 0\n");
#endif
		sum_f = (double) numerator_f/denominator_f;
	}

	void demote_f()
	{
		exponent_f = exponent_f + 1;
		long int den = (long int) pow (2, exponent_f);
		//printf("denominator = %ld\n", den);
		sub_frac_f(1, den);
	}

	void promote_f()
	{
		long int den = (long int) pow (2, exponent_f);
		//printf("denominator = %ld\n", den);
		add_frac_f(1, den);
		exponent_f = exponent_f - 1;
	}	

	//Reader
	void add_frac_r(long int num, long int den)
	{
		if(denominator_r%den == 0)
		{
			numerator_r = numerator_r + num * denominator_r/den;
		} else if (den%denominator_r == 0) {		
			numerator_r = numerator_r * den/denominator_r + num;
			denominator_r = den;
		} else {
			numerator_r = numerator_r * den + num * denominator_r;
			denominator_r = denominator_r * den;
		}
		sum_r = (double) numerator_r/denominator_r;
	}

	void sub_frac_r(long int num, long int den)
	{
		if(denominator_r%den == 0)
		{
			numerator_r = numerator_r - num * denominator_r/den;
		} else if (den%denominator_r == 0) {		
			numerator_r = numerator_r * den/denominator_r - num;
			denominator_r = den;
		} else {
			numerator_r = numerator_r * den - num * denominator_r;
			denominator_r = denominator_r * den;
		}
		sum_r = (double) numerator_r/denominator_r;
	}

	void demote_r()
	{
		exponent_r = exponent_r + 1;
		long int den = (long int) pow (2, exponent_r);
		//printf("denominator = %ld\n", den);
		sub_frac_r(1, den);
	}

	void promote_r()
	{
		long int den = (long int) pow (2, exponent_r);
		//printf("denominator = %ld\n", den);
		add_frac_r(1, den);
		exponent_r = exponent_r - 1;
	}	
}Item;

typedef struct Block
{
	long int start;
	long int finish;

	Block() 
	{
		start = 0;
		finish = 0;
	}
}Block;

/*class Comparator
{
public:
  Comparator()
    {}
  bool operator() (Method lhs, Method rhs)
  {
  	return (lhs.response > rhs.response);
  }
};*/

//FILE *pfile;

bool final_outcome;

unsigned int method_count;

bool fncomp (long int lhs, long int rhs) {return lhs < rhs;}

tbb::concurrent_queue<int> queue;
boost::lockfree::stack<int> stack(128);
tbb::concurrent_hash_map<int,int,MyHashCompare> map;

std::list<Method> thrd_lists[NUM_THRDS];

std::atomic<int> thrd_lists_size[NUM_THRDS];

std::atomic<bool> done[NUM_THRDS];

std::atomic<int> barrier(0);

void wait()
{
	barrier.fetch_add(1);
	while(barrier.load() < NUM_THRDS) { }
}

long int method_time[NUM_THRDS];
long int overhead_time[NUM_THRDS];

std::chrono::time_point<std::chrono::high_resolution_clock> start;

long int elapsed_time_verify;

void handle_failed_consumer(std::map<long int,Method,bool(*)(long int,long int)>& map_methods, std::unordered_map<int,Item>& map_items, std::map<long int,Method,bool(*)(long int,long int)>::iterator& it, std::unordered_map<int,Item>::iterator& it_item, std::stack<std::unordered_map<int,Item>::iterator>& stack_failed)
{
	std::map<long int,Method,bool(*)(long int,long int)>::iterator it_0;	
	for (it_0=map_methods.begin(); it_0 != it; ++it_0)
	{
#if LINEARIZABILITY
		if(it_0->second.response < it->second.invocation)
#elif SEQUENTIAL_CONSISTENCY
		if((it_0->second.response < it->second.invocation) && (it_0->second.process == it->second.process))
#elif SERIALIZABILITY
		if((it_0->second.response < it->second.invocation) && (it_0->second.txn_id == it->second.txn_id) || (it_0->second.txn_id < it->second.txn_id))
#endif
		{
			std::unordered_map<int,Item>::iterator it_item_0;
			it_item_0 = map_items.find(it_0->second.item_key);
			if(it_0->second.type == PRODUCER && it_item->second.status == PRESENT && (it_0->second.semantics == FIFO || it_0->second.semantics == LIFO || it->second.item_key == it_0->second.item_key))
			{
				//it_item->second.promote_failed.push(it_item_0->second.key);
				//it_item_0->second.demote_f();
				//printf("handle_failed\n");
				stack_failed.push(it_item_0);
			}
		}
	}
}

void handle_failed_read(std::map<long int,Method,bool(*)(long int,long int)>& map_methods, std::unordered_map<int,Item>& map_items, std::map<long int,Method,bool(*)(long int,long int)>::iterator& it, std::unordered_map<int,Item>::iterator& it_item, std::stack<std::unordered_map<int,Item>::iterator>& stack_failed)
{
	std::map<long int,Method,bool(*)(long int,long int)>::iterator it_0;	
	for (it_0=map_methods.begin(); it_0 != it; ++it_0)
	{
#if LINEARIZABILITY
		if(it_0->second.response < it->second.invocation)
#elif SEQUENTIAL_CONSISTENCY
		if((it_0->second.response < it->second.invocation) && (it_0->second.process == it->second.process))
#elif SERIALIZABILITY
		if((it_0->second.response < it->second.invocation) && (it_0->second.txn_id == it->second.txn_id) || (it_0->second.txn_id < it->second.txn_id))
#endif
		{
			std::unordered_map<int,Item>::iterator it_item_0;
			it_item_0 = map_items.find(it_0->second.item_key);
			if(it_0->second.type == PRODUCER && it_item->second.status == PRESENT && it->second.item_key == it_0->second.item_key)
			{
				//it_item->second.promote_failed.push(it_item_0->second.key);
				//it_item_0->second.demote_f();
				//printf("handle_failed_read\n");
				stack_failed.push(it_item_0);
			}
		}
	}
}

void verify_checkpoint(std::map<long int,Method,bool(*)(long int,long int)>& map_methods, std::unordered_map<int,Item>& map_items, std::map<long int,Method,bool(*)(long int,long int)>::iterator& it_start, long unsigned int& count_iterated, long int min, bool reset_it_start, std::map<long int,Block,bool(*)(long int,long int)>& map_block)
{
	if(!map_methods.empty())
	{
		//std::unordered_map<int,std::unordered_map<int,Item>::iterator> map_consumer;
		std::stack<std::unordered_map<int,Item>::iterator> stack_consumer;
		std::stack<std::map<long int,Method,bool(*)(long int,long int)>::iterator> stack_finished_methods;
		std::stack<std::unordered_map<int,Item>::iterator> stack_failed;

		//bool reset_it_start = true;
		std::map<long int,Method,bool(*)(long int,long int)>::iterator it;
		if(count_iterated == 0)
		{
			it=map_methods.begin();
			reset_it_start = false;
		//} else if(count_iterated < count_overall) {
		//} else if(count_iterated < map_methods.size()) {
		} else if(it != map_methods.end()) {
			it=++it_start;
		}	

	
		for (; it!=map_methods.end(); ++it)
		{
			if(it->second.response > min)
				break;

			/*if(it->second.invocation > min)
			{
				min = it->second.response;
				it_start = it;
				break;
			}*/
			//fprintf(pfile, "Checking method %d\n", it->second.id);
			
			if(method_count%5000 == 0)
				printf("Method Count = %u\n", method_count);
			method_count = method_count + 1;

			it_start = it;
			reset_it_start = false;
			count_iterated = count_iterated + 1;

			std::unordered_map<int,Item>::iterator it_item;
			// find item in map with it's key
			it_item = map_items.find(it->second.item_key);
			//(it_item->second).sum = (it_item->second).sum - 1;

#if DEBUG_
			if(it_item->second.status != PRESENT)
				printf("WARNING: Current item not present!\n");

			if((it->second).type == PRODUCER)
				printf("PRODUCER invocation %ld, response %ld, item %d\n", it->second.invocation, it->second.response, it->second.item_key);
			else if((it->second).type == CONSUMER)
				printf("CONSUMER invocation %ld, response %ld, item %d\n", it->second.invocation, it->second.response, it->second.item_key);
#endif

			if(it->second.type == PRODUCER)
			{
				it_item->second.producer = it;
	
				if(it_item->second.status == ABSENT)
				{
					//reset item parameters
					it_item->second.status = PRESENT;
					//printf("Clearing Item %d's demote_methods list\n", it_item->first);
					it_item->second.demote_methods.clear();
				}
				//(it_item->second).sum = (it_item->second).sum + 1;
				it_item->second.add_int(1);
		
				if(it->second.semantics == FIFO || it->second.semantics == LIFO)
				{
					std::map<long int,Method,bool(*)(long int,long int)>::iterator it_0;	
					for (it_0=map_methods.begin(); it_0 != it; ++it_0)
					{
	#if LINEARIZABILITY
						if(it_0->second.response < it->second.invocation)
	#elif SEQUENTIAL_CONSISTENCY
						if((it_0->second.response < it->second.invocation) && (it_0->second.process == it->second.process))
	#elif SERIALIZABILITY
						// if((it_0->second.response < it->second.invocation) && (it_0->second.txn_id == it->second.txn_id) || (it_0->second.txn_id < it->second.txn_id))
						if ((it_0->second.senderID == it->second.senderID) && it_0->second.requestAmnt > it->second.requestAmnt)
	#endif
						{
							std::unordered_map<int,Item>::iterator it_item_0;
							it_item_0 = map_items.find(it_0->second.item_key);
						
							//Demotion
							//FIFO Semantics
							if((it_0->second.type == PRODUCER && it_item_0->second.status == PRESENT) && (it->second.type == PRODUCER) && (it_0->second.semantics == FIFO))
							{
								it_item_0->second.promote_items.push(it_item->second.key);
								it_item->second.demote();
								it_item->second.demote_methods.push_back(it_0->second);
							} 
							
							//LIFO Semantics
							if((it_0->second.type == PRODUCER && it_item_0->second.status == PRESENT) && (it->second.type == PRODUCER) && (it_0->second.semantics == LIFO))
							{
								it_item->second.promote_items.push(it_item_0->second.key);
								it_item_0->second.demote();
								it_item_0->second.demote_methods.push_back(it->second);
								//printf("Pushing Item %d to Item %d's demote_method list\n", it->second.item_key, it_item_0->first);
							} 

						}
					}
				}
			}
			if(it->second.type == READER) //&& (it->second.semantics == SET || it->second.semantics == MAP)
			{
				if(it->second.status == true)
				{
					it_item->second.demote_r();

				} else {
					//TODO: Probably should handle the same as a failed consumer
					handle_failed_read(map_methods, map_items, it, it_item, stack_failed);
				}

			}

			//if(((it->second).type == CONSUMER) && ((it->second).semantics == FIFO))
			if(it->second.type == CONSUMER)
			{
				/*std::unordered_map<int,std::unordered_map<int,Item>::iterator>::iterator it_consumer;
				it_consumer = map_consumer.find((it->second).key);
				if(it_consumer == map_consumer.end())
				{
					std::pair<int,std::unordered_map<int,Item>::iterator> entry ((it->second).key,it);
					//map_consumer.insert(std::make_pair<int,std::unordered_map<int,Item>::iterator>((it->second).key,it));
					map_consumer.insert(entry);
				} else {
					it_consumer->second = it_item_0;
				}*/
				if(it->second.status == true)
				{
					//PROMOTE READS
					if(it_item->second.sum > 0)
					{
						it_item->second.sum_r = 0;
					}

					it_item->second.sub_int(1);

					it_item->second.status = ABSENT;

					// if(it_item->second.sum < 0)
					// {
					// 	//printf("WARNING: Sum < 0\n");
					// 	std::list<Method>::iterator it_method;
					// 	for(it_method = it_item->second.demote_methods.begin(); it_method != it_item->second.demote_methods.end(); ++it_method)
					// 	{
		
					// 		if((it->second.response < it_method->invocation) || (it_method->response < it->second.invocation))
					// 		{
					// 			//Methods do not overlap
					// 			//printf("NOTE: Methods do not overlap\n");
					// 		} else {
					// 			//printf("NOTE: CONSUME Item %d overlaps with PRODUCE Item %d\n", it->second.item_val, it_method->item_val);
					// 			it_item->second.promote();

					// 			//Need to remove from promote list
					// 			std::unordered_map<int,Item>::iterator it_method_item;
					// 			it_method_item = map_items.find(it_method->item_key);
					// 			std::stack<int> temp;
					// 			while(!it_method_item->second.promote_items.empty())
					// 			{
					// 				int top = it_method_item->second.promote_items.top();
					// 				if(top != it->second.item_key)
					// 				{
					// 					temp.push(top);
					// 				}
					// 				it_method_item->second.promote_items.pop();
					// 				//printf("Stuck here?\n");
					// 			}
					// 			it_method_item->second.promote_items.swap(temp);

					// 			it_method = it_item->second.demote_methods.erase(it_method);
					// 			--it_method;
								
					// 		}

					// 	}
					// }

					stack_consumer.push(it_item);

					//map_methods.erase(it); //TODO: Dangerous, make sure this is correct
					stack_finished_methods.push(it);
					//printf("stack_finished_methods.push(Consume(%d))\n", it->second.item_key);

					if(it_item->second.producer != map_methods.end())
					{
						stack_finished_methods.push(it_item->second.producer);
						//printf("stack_finished_methods.push(Produce(%d))\n", it_item->second.producer->second.item_key);
					}

				} else {	
					handle_failed_consumer(map_methods, map_items, it, it_item, stack_failed);
				}
			}
		}
		if(reset_it_start)
		{
			--it_start;
		}
		
		//NEED TO FLAG ITEMS ASSOCIATED WITH CONSUMER METHODS AS ABSENT
		while(!stack_consumer.empty())
		{
			std::unordered_map<int,Item>::iterator it_top = stack_consumer.top();

			//(it_top->second).status = ABSENT;
			
			//std::unordered_map<int,Item>::iterator it_top = it_consumer->second;
			//printf("promote_items.size() = %lu\n", (it_top->second).promote_items.size());

			//printf("Item %d: Promote item ", it_top->first);
			//std::list<int>::iterator it_promote;
			int item_promote;
			//for(it_promote = (it_top->second).promote_items.begin(); it_promote != (it_top->second).promote_items.end(); ++it_promote)
			while(!it_top->second.promote_items.empty())
			{
				//TODO: Check that item is not ABSENT
				item_promote = it_top->second.promote_items.top();
				std::unordered_map<int,Item>::iterator it_promote_item;
				it_promote_item = map_items.find(item_promote);
				//printf("%d ", item_promote);
				//(it_promote_item->second).sum = (it_promote_item->second).sum + 1;
				//(it_promote_item->second).add_int(1);
				//(it_promote_item->second).add_frac(1, 2);
				it_promote_item->second.promote();
				it_top->second.promote_items.pop();
			}
			//printf("\n");

			stack_consumer.pop();
		}	

		while(!stack_failed.empty())
		{
			std::unordered_map<int,Item>::iterator it_top = stack_failed.top();

			if(it_top->second.status == PRESENT)
			{
				it_top->second.demote_f();
			}
			
			stack_failed.pop();
		}

		//Remove methods that are no longer active
		
		while(!stack_finished_methods.empty())
		{
			std::map<long int,Method,bool(*)(long int,long int)>::iterator it_top = stack_finished_methods.top();
			map_methods.erase(it_top); //TODO: Dangerous, make sure this is correct
			stack_finished_methods.pop();
		}

		//Verify Sums
		std::unordered_map<int,Item>::iterator it_verify;

		bool outcome = true;
		for (it_verify = map_items.begin(); it_verify!=map_items.end(); ++it_verify)
		{
			if(it_verify->second.sum < 0)
			{
				outcome = false;
#if DEBUG_
				printf("WARNING: Item %d, sum %.2lf\n", it_verify->second.key, it_verify->second.sum);
#endif
			}
			//printf("Item %d, sum %.2lf\n", it_verify->second.key, it_verify->second.sum);

			if((ceil(it_verify->second.sum) + it_verify->second.sum_r) < 0)
			{
				outcome = false;
#if DEBUG_
				printf("WARNING: Item %d, sum_r %.2lf\n", it_verify->second.key, it_verify->second.sum_r);
#endif
			}

			int N;
			if(it_verify->second.sum_f == 0)
				N = 0;
			else
				N = -1;

			if(((ceil(it_verify->second.sum) + it_verify->second.sum_f) * N) < 0)
			{
				outcome = false;
#if DEBUG_
				printf("WARNING: Item %d, sum_f %.2lf\n", it_verify->second.key, it_verify->second.sum_f);
#endif
			}
		}
		if(outcome == true)
		{
			final_outcome = true;
#if DEBUG_
			printf("-------------Program Correct Up To This Point-------------\n");
#endif
		} else
		{
			final_outcome = false;
#if DEBUG_
			printf("-------------Program Not Correct-------------\n");
#endif
		}

		
	}
}

void work_queue(int id)
{
	
	//int keyRange = 20;
	unsigned int testSize = TEST_SIZE;

	double wallTime = 0.0;
    
	timeval tod;
	gettimeofday(&tod,0);
	wallTime += static_cast<double>(tod.tv_sec);
	wallTime += static_cast<double>(tod.tv_usec) * 1e-6;

    //boost::mt19937 randomGenKey;
    boost::mt19937 randomGenOp;
    //randomGenKey.seed(wallTime + id);
    randomGenOp.seed(wallTime + id + 1000);
    //boost::uniform_int<int> randomDistKey(1, keyRange);
    boost::uniform_int<unsigned int> randomDistOp(1, 100);


	auto start_time = std::chrono::time_point_cast<std::chrono::nanoseconds>(start);
	auto start_time_epoch = start_time.time_since_epoch();

	//int m_id = id;
	int m_id = id + 1;
	
	//printf("Spawning thread %d, start = %ld\n", id, start_time_epoch.count());
	
	std::chrono::time_point<std::chrono::high_resolution_clock> end;

	wait();
	for(unsigned int i = 0; i < testSize; i++) 
	{
		Type type;
		int item_key = -1;
		bool res = true;
		
		uint32_t op_dist = randomDistOp(randomGenOp);

		end = std::chrono::high_resolution_clock::now();

		auto pre_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		auto pre_function_epoch = pre_function.time_since_epoch();

		//auto invocation =
		//std::chrono::duration_cast<std::chrono::nanoseconds>(end - start);
		//printf("Thread %d invocation: %ld nanoseconds\n", id, invocation.count());

		long int invocation = pre_function_epoch.count() - start_time_epoch.count();

		//if(invocation.count() > (LONG_MAX - 10000000000)) //10 seconds until time exceeds limit
		if(invocation > (LONG_MAX - 10000000000)) //10 seconds until time exceeds limit
		{
#if DEBUG_
			printf("WARNING: TIME LIMIT REACHED! TERMINATING PROGRAM\n");
#endif
			break;
		}
		
		if(op_dist <= 50)
		//if(op_dist <= 33)
		{
			type = CONSUMER;
			int item_pop;
			long unsigned int* item_pop_ptr;

			res = queue.try_pop (item_pop);
			if(res)
			{
				item_key = item_pop;
			} else
				item_key = INT_MIN;
		} else {
			type = PRODUCER;
			//item_key = randomDistKey(randomGenKey);
			item_key = m_id;
			queue.push(item_key);
		}
				
		end = std::chrono::high_resolution_clock::now();

		auto post_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		auto post_function_epoch = post_function.time_since_epoch();
		
		//printf("Thread %d, item %d: prefunction = %ld, postfunction = %ld\n", id, item_key, pre_function_epoch.count(), post_function_epoch.count());

		//auto response =
		//std::chrono::duration_cast<std::chrono::nanoseconds>(end - start);

		long int response = post_function_epoch.count() - start_time_epoch.count();

		//printf("Thread %d response: %ld nanoseconds\n", id, response.count());
		
		/*Method m1;
		m1.id = m_id;
		m1.process = id;

		m1.item_key = item_key;
		m1.type = type;
		m1.semantics = FIFO;
		m1.invocation = invocation.count();
		m1.response = response.count();*/

		//Method m1(m_id, id, item_key, FIFO, type, invocation.count(), response.count(), res);
		Method m1(m_id, id, item_key, INT_MIN, FIFO, type, invocation, response, res, m_id);
		
		m_id = m_id + NUM_THRDS;
		
		thrd_lists[id].push_back(m1);
		
		thrd_lists_size[id].fetch_add(1);
		
		//method_time[id] = method_time[id] + (response.count() - invocation.count());
		method_time[id] = method_time[id] + (response - invocation);
		
	}
	
	done[id].store(true);
}


void work_stack(int id)
{

	//int keyRange = 20;
	unsigned int testSize = TEST_SIZE;

	double wallTime = 0.0;
    
	timeval tod;
	gettimeofday(&tod,0);
	wallTime += static_cast<double>(tod.tv_sec);
	wallTime += static_cast<double>(tod.tv_usec) * 1e-6;

    //boost::mt19937 randomGenKey;
    boost::mt19937 randomGenOp;
    //randomGenKey.seed(wallTime + id);
    randomGenOp.seed(wallTime + id + 1000);
    //boost::uniform_int<int> randomDistKey(1, keyRange);
    boost::uniform_int<unsigned int> randomDistOp(1, 100);
	
	auto start_time = std::chrono::time_point_cast<std::chrono::nanoseconds>(start);
	auto start_time_epoch = start_time.time_since_epoch();

	int m_id = id;
	
	std::chrono::time_point<std::chrono::high_resolution_clock> end;

	wait();
	for(unsigned int i = 0; i < testSize; i++)
	{
		Type type;

		int item_key = -1;

		bool res = true;

		uint32_t op_dist = randomDistOp(randomGenOp);
		
		end = std::chrono::high_resolution_clock::now();

		auto pre_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		auto pre_function_epoch = pre_function.time_since_epoch();

		//auto invocation =
		//std::chrono::duration_cast<std::chrono::nanoseconds>(end - start);
		//printf("Thread %d invocation: %ld nanoseconds\n", id, invocation.count());

		long int invocation = pre_function_epoch.count() - start_time_epoch.count();

		//if(invocation.count() > (LONG_MAX - 10000000000)) //10 seconds until time exceeds limit
		if(invocation > (LONG_MAX - 10000000000)) //10 seconds until time exceeds limit
		{
#if DEBUG_
			printf("WARNING: TIME LIMIT REACHED! TERMINATING PROGRAM\n");
#endif
			break;
		}
		
		if(op_dist <= 50)
		{	
			type = CONSUMER;
			int item_pop;
			res = stack.pop (item_pop);
			if(res)
				item_key = item_pop;
			else
				item_key = INT_MIN;
			
		} else {
			type = PRODUCER;
			item_key = m_id;
			stack.push(item_key);
		}
				
		end = std::chrono::high_resolution_clock::now();

		auto post_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		auto post_function_epoch = post_function.time_since_epoch();
		
		//printf("Thread %d, item %d: prefunction = %ld, postfunction = %ld\n", id, item_key, pre_function_epoch.count(), post_function_epoch.count());

		//auto response =
		//std::chrono::duration_cast<std::chrono::nanoseconds>(end - start);
		//printf("Thread %d response: %ld nanoseconds\n", id, response.count());

		long int response = post_function_epoch.count() - start_time_epoch.count();
		
		/*Method m1;
		m1.id = m_id;

		m1.item_key = item_key;
		m1.process = id;

		m1.type = type;
		//m1.semantics = FIFO;

		m1.semantics = LIFO;
		m1.invocation = invocation.count();

		m1.response = response.count();*/

		//Method m1(m_id, id, item_key, LIFO, type, invocation.count(), response.count(), res);
		Method m1(m_id, id, item_key, INT_MIN, LIFO, type, invocation, response, res, m_id);
			
		m_id = m_id + NUM_THRDS;
		
		thrd_lists[id].push_back(m1);
		
		thrd_lists_size[id].fetch_add(1);
		
		//method_time[id] = method_time[id] + (response.count() - invocation.count());
		method_time[id] = method_time[id] + (response - invocation);
		
	}
	
	done[id].store(true);
}

void work_map(int id)
{

	//int keyRange = 20;
	unsigned int testSize = TEST_SIZE;

	double wallTime = 0.0;
    
	timeval tod;
	gettimeofday(&tod,0);
	wallTime += static_cast<double>(tod.tv_sec);
	wallTime += static_cast<double>(tod.tv_usec) * 1e-6;

    //boost::mt19937 randomGenKey;
    boost::mt19937 randomGenOp;
    //randomGenKey.seed(wallTime + id);
    randomGenOp.seed(wallTime + id + 1000);
    //boost::uniform_int<int> randomDistKey(1, keyRange);
    boost::uniform_int<unsigned int> randomDistOp(1, 100);

	auto start_time = std::chrono::time_point_cast<std::chrono::nanoseconds>(start);
	auto start_time_epoch = start_time.time_since_epoch();

	int m_id = id;
	
	std::chrono::time_point<std::chrono::high_resolution_clock> end;

	wait();
	for(unsigned int i = 0; i < testSize; i++) 
	{
		Type type;

		int item_key = -1;

		int item_val = -1;

		bool res = true;

		uint32_t op_dist = randomDistOp(randomGenOp);
		
		end = std::chrono::high_resolution_clock::now();

		auto pre_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		auto pre_function_epoch = pre_function.time_since_epoch();

		//auto invocation =
		//std::chrono::duration_cast<std::chrono::nanoseconds>(end - start);
		//printf("Thread %d invocation: %ld nanoseconds\n", id, invocation.count());

		long int invocation = pre_function_epoch.count() - start_time_epoch.count();

		//if(invocation.count() > (LONG_MAX - 10000000000)) //10 seconds until time exceeds limit
		if(invocation > (LONG_MAX - 10000000000)) //10 seconds until time exceeds limit
		{
#if DEBUG_
			printf("WARNING: TIME LIMIT REACHED! TERMINATING PROGRAM\n");
#endif
			break;
		}

//if (TBB_MAP) {
		tbb::concurrent_hash_map<int,int,MyHashCompare>::accessor a;
//} 
		if(op_dist <= 33)
		{
			type = CONSUMER;
			int item_erase = m_id - 2*NUM_THRDS;
			res = map.erase( item_erase );
			if(res)
			{
				item_key = item_erase;
			} else {
				item_key = INT_MIN;	
			}

		} else if (op_dist <= 66) {
			type = PRODUCER;
			item_key = m_id;
			item_val = m_id;
			map.insert(a, item_key);
			a->second = item_val;

		} else {
			type = READER;
			item_key = m_id - NUM_THRDS;
			res = map.find(a, item_key);
			if(res)
			{
				item_val = a->second;
			} else {
				item_key = INT_MIN;
				item_val = INT_MIN;
			}
		}
				
		end = std::chrono::high_resolution_clock::now();

		auto post_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		auto post_function_epoch = post_function.time_since_epoch();
		
		//printf("Thread %d, item %d: prefunction = %ld, postfunction = %ld\n", id, item_key, pre_function_epoch.count(), post_function_epoch.count());

		//auto response =
		//std::chrono::duration_cast<std::chrono::nanoseconds>(end - start);
		//printf("Thread %d response: %ld nanoseconds\n", id, response.count());

		long int response = post_function_epoch.count() - start_time_epoch.count();
		
		/*Method m1;
		m1.id = m_id;
		m1.item_key = item_key;
		m1.process = id;
		m1.type = type;
		//m1.semantics = FIFO;
		m1.semantics = LIFO;
		m1.invocation = invocation.count();
		m1.response = response.count();*/

		//Method m1(m_id, id, item_key, LIFO, type, invocation.count(), response.count(), res);
		Method m1(m_id, id, item_key, item_val, MAP, type, invocation, response, res, m_id);
			
		m_id = m_id + NUM_THRDS;
		
		thrd_lists[id].push_back(m1);
		
		thrd_lists_size[id].fetch_add(1);
		
		//method_time[id] = method_time[id] + (response.count() - invocation.count());
		method_time[id] = method_time[id] + (response - invocation);
		
	}
	
	done[id].store(true);
}

void verify()
{
	wait();
	auto start_time = std::chrono::time_point_cast<std::chrono::nanoseconds>(start);
	auto start_time_epoch = start_time.time_since_epoch();

	std::chrono::time_point<std::chrono::high_resolution_clock> end;

	end = std::chrono::high_resolution_clock::now();

	auto pre_verify = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
	auto pre_verify_epoch = pre_verify.time_since_epoch();

	long int verify_start = pre_verify_epoch.count() - start_time_epoch.count();

	//std::priority_queue<Method,std::vector<Method>,Comparator> pq_methods;
	bool(*fn_pt)(long int,long int) = fncomp;
  	std::map<long int,Method,bool(*)(long int,long int)> map_methods (fn_pt);
	std::map<long int,Block,bool(*)(long int,long int)> map_block (fn_pt);

	std::unordered_map<int,Item> map_items;

	std::map<long int,Method,bool(*)(long int,long int)>::iterator it_start;

	std::list<Method>::iterator it[NUM_THRDS];

	int it_count[NUM_THRDS];

	bool stop = false;

	long int min; 

	long unsigned int count_overall = 0;

	long unsigned int count_iterated = 0;

	//min = LONG_MAX;

	long int old_min;

	std::map<long int,Method,bool(*)(long int,long int)>::iterator it_qstart;


	while(!stop)
	{
		stop = true;
		min = LONG_MAX;

		for(int i = 0; i < NUM_THRDS; i++)
		{
			if(done[i].load() == false)
			{
				stop = false;
			}
			
			long int response_time = 0;
			while(it_count[i] < thrd_lists_size[i].load())
			{
				if(it_count[i] == 0)
				{
					it[i] = thrd_lists[i].begin();
				} else {
					++it[i];
				}
				
				Method m = *it[i];

				//if(m.item_key%500 == 0)
					//printf("Checking method %d\n", m.item_key);

				//pq_methods.push(m);

				//Consider doing a left shift by bits to store number of threads, and store thread id in lower bits
				std::map<long int,Method,bool(*)(long int,long int)>::iterator it_method;
				it_method = map_methods.find(m.response);
  				while (it_method != map_methods.end())
				{
					m.response = m.response + 1;
					it_method = map_methods.find(m.response);
				}
				response_time = m.response;

				map_methods.insert ( std::pair<long int,Method>(m.response,m) );

				//printf("Insert method %d\n", m.id);
				
				it_count[i] = it_count[i] + 1;

				count_overall = count_overall + 1;

				std::unordered_map<int,Item>::iterator it_item;
				it_item = map_items.find(m.item_key);
				if(it_item == map_items.end())
				{
					/*Item item;
					item.key = m.item_key;
					item.sum = 0;
					item.numerator = 0;
					item.denominator = 1;
					item.exponent = 0;
					item.status = PRESENT;*/
					Item item(m.item_key);
					item.producer = map_methods.end();
					map_items.insert(std::pair<int,Item>(m.item_key,item) );
					it_item = map_items.find(m.item_key);
				}
				
				/*if(m.type == PRODUCER)
				{
					//TODO: If Item is ABSENT, set it to PRESENT
					if(it_item->second.status == ABSENT)
					{
						it_item->second.status = PRESENT;
					}
					//(it_item->second).sum = (it_item->second).sum + 1;
					it_item->second.add_int(1);
					
				} else if(m.type == CONSUMER)
				{
					//(it_item->second).sum = (it_item->second).sum - 1;
					it_item->second.sub_int(1);
				}*/
			}	

			if(response_time < min)
			{
				min = response_time;
			}
		}
		//printf("min = %ld\n", min);

		verify_checkpoint(map_methods, map_items, it_start, count_iterated, min, true, map_block);
		/*if(min == old_min)
		{
			min = LONG_MAX;
		}*/

	}

	verify_checkpoint(map_methods, map_items, it_start, count_iterated, LONG_MAX, false, map_block);

#if DEBUG_
	printf("Count overall = %lu, count iterated = %lu, map_methods.size() = %lu\n", count_overall, count_iterated, map_methods.size());
#endif

#if DEBUG_
	printf("All threads finished!\n");

	/*std::map<long int,Method,bool(*)(long int,long int)>::iterator it_m;
	for (it_m=map_methods.begin(); it_m!=map_methods.end(); ++it_m)
	{
		set_quiescent_period(map_methods, map_block, it_m);
	}*/

	std::map<long int,Block,bool(*)(long int,long int)>::iterator it_b;
	for (it_b=map_block.begin(); it_b!=map_block.end(); ++it_b)
	{
		printf("Block start = %ld, finish = %ld\n", it_b->second.start, it_b->second.finish);
	}


	std::map<long int,Method,bool(*)(long int,long int)>::iterator it_;
	for (it_=map_methods.begin(); it_!=map_methods.end(); ++it_)
	{
		std::unordered_map<int,Item>::iterator it_item;
		it_item = map_items.find(it_->second.item_key);
		if(it_->second.type == PRODUCER)
			printf("PRODUCER inv %ld, res %ld, item %d, sum %.2lf, sum_r = %.2lf, sum_f = %.2lf, tid = %d, qperiod = %d\n", it_->second.invocation, it_->second.response, it_->second.item_key, it_item->second.sum, it_item->second.sum_r, it_item->second.sum_f, it_->second.process, it_->second.quiescent_period);
		else if ((it_->second).type == CONSUMER)
			printf("CONSUMER inv %ld, res %ld, item %d, sum %.2lf, sum_r = %.2lf, sum_f = %.2lf, tid = %d, qperiod = %d\n", it_->second.invocation, it_->second.response, it_->second.item_key, it_item->second.sum, it_item->second.sum_r, it_item->second.sum_f, it_->second.process, it_->second.quiescent_period);
		else if ((it_->second).type == READER)
			printf("READER inv %ld, res %ld, item %d, sum %.2lf, sum_r = %.2lf, sum_f = %.2lf, tid = %d, qperiod = %d\n", it_->second.invocation, it_->second.response, it_->second.item_key, it_item->second.sum, it_item->second.sum_r, it_item->second.sum_f, it_->second.process, it_->second.quiescent_period);
	}
#endif

	end = std::chrono::high_resolution_clock::now();

	auto post_verify = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
	auto post_verify_epoch = post_verify.time_since_epoch();
	long int verify_finish = post_verify_epoch.count() - start_time_epoch.count();

	elapsed_time_verify = verify_finish - verify_start;
}

int main(int argc,char* argv[]) 
{ 
	//pfile = fopen("output.txt", "a");

	method_count = 0;

	TBB_QUEUE = 0;
	BOOST_STACK = 0;
	TBB_MAP = 0;
	if( argc == 2 ) {
		printf("Test size = %d\n", atoi(argv[1]));
		TEST_SIZE = (unsigned int) atoi(argv[1]);
		TBB_QUEUE = 1;
		printf("Testing TBB_QUEUE\n");
	} else if (argc == 3) {
		printf("Test size = %d\n", atoi(argv[1]));
		TEST_SIZE = (unsigned int) atoi(argv[1]);
		if(atoi(argv[2]) == 0)
		{
			TBB_QUEUE = 1;
			printf("Testing TBB_QUEUE\n");
		} else if(atoi(argv[2]) == 1)
		{
			BOOST_STACK = 1;
			printf("Testing BOOST_STACK\n");
		} else if(atoi(argv[2]) == 2)
		{
			TBB_MAP = 1;
			printf("Testing TBB_MAP\n");
		} 
		
	} else { //default
		printf("Test size = 10\n");
		TEST_SIZE = 10;
		TBB_QUEUE = 1;
		printf("Testing TBB_QUEUE\n");
	}

	final_outcome = true;

	std::thread t[NUM_THRDS];
	
	std::thread v;

	start = std::chrono::high_resolution_clock::now();

	// generate work to verify
	for(int i = 0; i < NUM_THRDS; i++)
	{
		if(TBB_QUEUE)
		{
			t[i] = std::thread(work_queue,i);
		} else if(BOOST_STACK)
		{
			t[i] = std::thread(work_stack,i);
		} else if(TBB_MAP)
		{
			t[i] = std::thread(work_map,i);
		} 
		//t[i] = std::thread(test_map,i);
	}
	
	v = std::thread(verify);
	
	for(int i = 0; i < NUM_THRDS; i++)
	{
		t[i].join();
	}
	//TODO: Uncomment line 1426
	v.join();

	if(final_outcome == true)
	{
		printf("-------------Program Correct Up To This Point-------------\n");
	} else {
		printf("-------------Program Not Correct-------------\n");
	}
	
	auto finish = std::chrono::high_resolution_clock::now();
	auto elapsed_time = std::chrono::duration_cast<std::chrono::nanoseconds>(finish - start);
	double elapsed_time_double = elapsed_time.count()*0.000000001;
	printf("Total Time: %.15lf seconds\n", elapsed_time_double);
	//auto elapsed_time = std::chrono::duration_cast<std::chrono::seconds>(finish - start);
	//std::cout << elapsed_time.count() << std::endl;
	
	long int elapsed_time_method = 0;
	long int elapsed_overhead_time = 0;
	
	for(int i = 0; i < NUM_THRDS; i++)
	{
		if(method_time[i] > elapsed_time_method)
			elapsed_time_method = method_time[i];
		if(overhead_time[i] > elapsed_overhead_time)
			elapsed_overhead_time = overhead_time[i];
		//elapsed_time_method = elapsed_time_method + method_time[i];
		//elapsed_overhead_time = elapsed_overhead_time + overhead_time[i];
	}
	
	double elapsed_time_method_double = elapsed_time_method*0.000000001;
	double elapsed_overhead_time_double = elapsed_overhead_time*0.000000001;
	double elapsed_time_verify_double = elapsed_time_verify*0.000000001;
	
	printf("Total Method Time: %.15lf seconds\n", elapsed_time_method_double);
	printf("Total Overhead Time: %.15lf seconds\n", elapsed_overhead_time_double);
	
	//double elapsed_time_verify_double = elapsed_time_double - elapsed_time_method_double;
	//double elapsed_time_verify_double = elapsed_time_double - elapsed_time_method_double - elapsed_overhead_time_double;
	elapsed_time_verify_double = elapsed_time_verify_double - elapsed_time_method_double;
	
	printf("Total Verification Time: %.15lf seconds\n", elapsed_time_verify_double);

	
	//fprintf(pfile, "%.15lf %.15lf\n", elapsed_time_method_double, elapsed_time_verify_double);
	//fclose(pfile);
	
if (TBB_QUEUE) {	
	printf("Final Queue Configuration:\n");
	typedef tbb::concurrent_queue<int>::iterator iter;
    for(iter i(queue.unsafe_begin()); i!=queue.unsafe_end(); i++)
        printf("%d ", *i);
    printf("\n");
} else if (BOOST_STACK) {
	printf("Final Stack Configuration:\n");
	int stack_val;
	while(stack.pop(stack_val))
	{
		printf("%d ", stack_val);
	}
	printf("\n");
} else if (TBB_MAP) {
	printf("Final Map Configuration:\n");
	tbb::concurrent_hash_map<int,int,MyHashCompare>::iterator it;
	for( it=map.begin(); it!=map.end(); ++it )
    	printf("%d,%d ",it->first,it->second);
	printf("\n");
}
    return 0; 
} 
