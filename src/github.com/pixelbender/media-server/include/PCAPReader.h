
#ifndef PCAPMEDIAFILE_H
#define PCAPMEDIAFILE_H

#include "log.h"
#include "rtp.h"

class PCAPReader
{
public:
	PCAPReader();
	bool Open(const char* file);
	RTPPacket::shared GetNextPacket(const RTPMap& rtpMap,const RTPMap& extMap);
	uint64_t Seek(const uint64_t time);
	void Rewind();
	bool Close();

private:
	uint8_t data[65535];
	size_t size = 65535;
	int fd;
	size_t len;
};

#endif /* PCAPMEDIAFILE_H */

