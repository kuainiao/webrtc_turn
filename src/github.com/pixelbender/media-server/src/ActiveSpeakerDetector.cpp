#include "ActiveSpeakerDetector.h"

#include <algorithm>
#include <log.h>

static const uint64_t ScorePerMiliScond = 10;
static const uint64_t MaxScore = 6000;
static const uint64_t MinInterval = 10;

void ActiveSpeakerDetector::Accumulate(uint32_t id, bool vad, uint8_t db, uint64_t now)
{
	//Search for the speacker
	auto it = speakers.find(id);

	//Check if we had that speakcer before
	if (it==speakers.end())
	{
		//Store 1s of initial bump if vad
		speakers[id] = {vad ? ScorePerMiliScond*1000ul : 0ul,now};
	} 
	//Accumulate only if audio has been detected
	else if (vad)
	{
		// The audio level is expressed in -dBov, with values from 0 to 127
		// representing 0 to -127 dBov. dBov is the level, in decibels, relative
		// to the overload point of the system.
		// The audio level for digital silence, for example for a muted audio
		// source, MUST be represented as 127 (-127 dBov).
		WORD level = 64 + (127-db)/2;

		//Get time diff from last score, we consider 1s as max to coincide with initial bump
		uint64_t diff = std::min(now-it->second.ts,(uint64_t)1000ul);
		//UltraDebug("-ActiveSpeakerDetector::Accumulate [id:%u,vad:%d,diff:%u,level:%u,score:%lu]\n",id,vad,diff,level,speakers[id].score);
		//Do not accumulate too much so we can switch faster
		it->second.score = std::min(it->second.score+diff*level/ScorePerMiliScond,MaxScore);
		//Set last update time
		it->second.ts = now;
	}
	
	//UltraDebug("-ActiveSpeakerDetector::Accumulate [id:%u,vad:%d,dbs:%u,score:%lu]\n",id,vad,db,speakers[id].score);
	
	//Process vads and check new 
	Process(now);
	
}

void ActiveSpeakerDetector::Release(uint32_t id)
{
	//Remove speaker
	speakers.erase(id);
}

void ActiveSpeakerDetector::Process(uint64_t now)
{
	uint64_t active = 0;
	uint64_t maxScore = 0;
	//Get difference from last process
	uint64_t diff = now-last;
	
	//If we have processed it quite recently
	if (diff<MinInterval)
		return;
	
	//Reduce accumulated voice activity
	uint64_t decay = diff*ScorePerMiliScond;
	
	//For each 
	for (auto& entry : speakers)
	{
		//Decay
		if (entry.second.score>decay)
			//Decrease score
			entry.second.score -= decay;
		else
			//None
			entry.second.score = 0;
		
		//UltraDebug("-ActiveSpeakerDetector::Process [id:%u,score:%llu,decay:%llu]\n",entry.first,entry.second.score,decay);
		
		//Check if it is active speaker
		if (maxScore<entry.second.score)
		{
			//New active
			active = entry.first;
			maxScore = entry.second.score;
		}
	}
	
	//IF active has changed and we are out of the block period
	if (active!=lastActive && now>blockedUntil)
	{
		//Event
		listener->onActiveSpeakerChanded(active);
		//Store last aceive and calculate blocking time
		lastActive = active;
		blockedUntil = now + minChangePeriod;
		//UltraDebug("-ActiveSpeakerDetector::Process [active:%u,blockedUntil:%ull]\n",active,blockedUntil);
	}
	
	//Update laste process time
	last = now;
}
