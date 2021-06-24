/*
 * To change this license header, choose License Headers in Project Properties.
 * To change this template file, choose Tools | Templates
 * and open the template in the editor.
 */

/* 
 * File:   PCAPTransportEmulator.h
 * Author: Sergio
 *
 * Created on 12 de julio de 2018, 15:45
 */

#ifndef PCAPTRANSPORTEMULATOR_H
#define PCAPTRANSPORTEMULATOR_H

#include "PCAPReader.h"


class PCAPTransportEmulator : 
	public RTPReceiver
{
public:
	PCAPTransportEmulator();
	virtual ~PCAPTransportEmulator();
	
	void SetRemoteProperties(const Properties& properties);

	bool AddIncomingSourceGroup(RTPIncomingSourceGroup *group);
	bool RemoveIncomingSourceGroup(RTPIncomingSourceGroup *group);
	
	bool Open(const char* filename);
	bool Play();
	uint64_t Seek(uint64_t time);
	bool Stop();
	bool Close();
	
	// RTPReceiver interface
	virtual int SendPLI(DWORD ssrc) override { return 1; }
	
private:
	int Run();
	
private:
	PCAPReader reader;
	
	RTPMap		rtpMap;
	RTPMap		extMap;
	RTPMap		aptMap;
	std::map<DWORD,RTPIncomingSourceGroup*> incoming;
	uint64_t first		= 0;
	pthread_t thread	= {0};
	bool running		= false;;
	WaitCondition cond;
};

#endif /* PCAPTRANSPORTEMULATOR_H */

