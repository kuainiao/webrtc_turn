/*
 * To change this license header, choose License Headers in Project Properties.
 * To change this template file, choose Tools | Templates
 * and open the template in the editor.
 */

/* 
 * File:   VP9LayerSelector.cpp
 * Author: Sergio
 * 
 * Created on 1 de febrero de 2017, 21:21
 */

#include "VP9LayerSelector.h"
#include "VP9PayloadDescription.h"

VP9LayerSelector::VP9LayerSelector()
{
	temporalLayerId		= 0;
	spatialLayerId		= 0;
	nextTemporalLayerId	= LayerInfo::MaxLayerId;
	nextSpatialLayerId	= LayerInfo::MaxLayerId;
}

VP9LayerSelector::VP9LayerSelector(BYTE temporalLayerId,BYTE spatialLayerId )
{
	temporalLayerId		= 0;
	spatialLayerId		= 0;
	nextTemporalLayerId	= temporalLayerId;
	nextSpatialLayerId	= spatialLayerId;
}

void VP9LayerSelector::SelectTemporalLayer(BYTE id)
{
	//Set next
	nextTemporalLayerId = id;
}

void VP9LayerSelector::SelectSpatialLayer(BYTE id)
{
	//Set next
	nextSpatialLayerId = id;
}
	
bool VP9LayerSelector::Select(const RTPPacket::shared& packet,bool &mark)
{
	VP9PayloadDescription desc = {};
	
	//If packet has frame markings
	if (packet->HasFrameMarkings())
	{
		//Get it from frame markings
		const auto& fm = packet->GetFrameMarks();
		//Import data
		desc.pictureIdPresent			= false;
		desc.layerIndicesPresent		= 0;
		desc.flexibleMode			= 0;
		desc.startOfLayerFrame			= fm.startOfFrame;
		desc.endOfLayerFrame			= fm.endOfFrame;
		desc.scalabiltiyStructureDataPresent	= 0;
		desc.pictureId				= 0;
		desc.temporalLayerId			= fm.temporalLayerId;
		// The following  shows VP9 Layer encoding information (3 bits for
		// spatial and temporal layer) mapped to the generic LID and TID fields.
		// The P and U bits MUST match the corresponding bits in the VP9 Payload
		// Description.
		//    0                
		//    0 1 2 3 4 5 6 7
		//   +-+-+-+-+-+-+-+-+
		//   |0|0|0|P|U| SID |
		//   +-+-+-+-+-+-+-+-+
		desc.interPicturePredictedLayerFrame	= fm.layerId & 0x20;
		desc.switchingPoint			= fm.layerId & 0x10;
		desc.spatialLayerId			= fm.layerId & 0x07;
		desc.interlayerDependencyUsed		= false;
		desc.temporalLayer0Index		= fm.tl0PicIdx;

	//Parse VP9 payload description
	} else if (!desc.Parse(packet->GetMediaData(),packet->GetMaxMediaLength()))
		//Error
		return Error("-VP9LayerSelector::Select() | Cannot parse VP9PayloadDescription\n");
	
	//if (desc.startOfLayerFrame)
	//	UltraDebug("-VP9LayerSelector::Select() | #%d T%dS%d P=%d D=%d S=%d %s\n", desc.pictureId-42,desc.temporalLayerId,desc.spatialLayerId,desc.interPicturePredictedLayerFrame,desc.interlayerDependencyUsed,desc.switchingPoint
	//		,desc.interPicturePredictedLayerFrame==0 && desc.spatialLayerId==1 ? "<----------------------":"");
	
	//Store current temporal id
	BYTE currentTemporalLayerId = temporalLayerId;
	
	//Check if we need to upscale temporally
	if (nextTemporalLayerId>temporalLayerId)
	{
		//Check if we can upscale and it is the start of the layer and it is a valid layer
		if (desc.switchingPoint && desc.startOfLayerFrame && desc.temporalLayerId<=nextTemporalLayerId)
		{
			//UltraDebug("-VP9LayerSelector::Select() | Upscaling temporalLayerId [id:%d,target:%d]\n",desc.temporalLayerId,nextTemporalLayerId);
			//Update current layer
			temporalLayerId = desc.temporalLayerId;
			currentTemporalLayerId = temporalLayerId;
		}
	//Check if we need to downscale
	} else if (nextTemporalLayerId<temporalLayerId) {
		//We can only downscale on the end of a layer to set the market bit
		if (desc.endOfLayerFrame)
		{
			//UltraDebug("-VP9LayerSelector::Select() | Downscaling temporalLayerId [id:%d,target:%d]\n",temporalLayerId,nextTemporalLayerId);
			//Update to target layer for next packets
			temporalLayerId = nextTemporalLayerId;
		}
	}
	
	//If it is from the current layer
	if (currentTemporalLayerId<desc.temporalLayerId)
	{
		//UltraDebug("-VP9LayerSelector::Select() | dropping packet based on temporalLayerId [us:%d,desc:%d,mark:%d]\n",temporalLayerId,desc.temporalLayerId,packet->GetMark());
		//Drop it
		return false;
	}
	
	//Get current spatial layer
	BYTE currentSpatialLayerId = spatialLayerId;
	
	//Check if we need to upscale spatially
	if (nextSpatialLayerId>spatialLayerId)
	{
		/*
			Inter-picture predicted layer frame.  When set to zero, the layer
			frame does not utilize inter-picture prediction.  In this case,
			up-switching to current spatial layer's frame is possible from
			directly lower spatial layer frame.  P SHOULD also be set to zero
			when encoding a layer synchronization frame in response to an LRR
		 */
		//Check if we can upscale and it is the start of the layer and it is a valid layer
		if (desc.interPicturePredictedLayerFrame==0 && desc.startOfLayerFrame && desc.spatialLayerId<nextSpatialLayerId)
		{
			//Update current layer
			spatialLayerId = desc.spatialLayerId+1;
			//UltraDebug("-VP9LayerSelector::Select() | Upscaling spatialLayerId [id:%d,to:%d,target:%d]\n",desc.spatialLayerId,spatialLayerId,nextSpatialLayerId);
			
		}
	//Ceck if we need to downscale
	} else if (nextSpatialLayerId<spatialLayerId) {
		//We can only downscale on the end of a layer to set the market bit
		if (desc.endOfLayerFrame)
		{
			//UltraDebug("-VP9LayerSelector::Select() | Downscaling spatialLayerId [id:%d,target:%d]\n",spatialLayerId,nextSpatialLayerId);
			//Update to target layer
			spatialLayerId = nextSpatialLayerId;
		}
	}
	
	//If it is not valid for the current layer
	if (currentSpatialLayerId<desc.spatialLayerId)
	{
		//UltraDebug("-VP9LayerSelector::Select() | dropping packet based on spatialLayerId [us:%d,desc:%d,mark:%d]\n",spatialLayerId,desc.spatialLayerId,packet->GetMark());
		//Drop it
		return false;
	}
	
	//RTP mark is set for the last frame layer of the selected layer
	mark = packet->GetMark() || (desc.endOfLayerFrame && spatialLayerId==desc.spatialLayerId);
	
	//UltraDebug("-VP9LayerSelector::Select() | Accepting packet [extSegNum:%u,mark:%d,tid:%d,sid:%d]\n",packet->GetExtSeqNum(),mark,desc.temporalLayerId,desc.spatialLayerId);
	//Select
	return true;
	
}

 LayerInfo VP9LayerSelector::GetLayerIds(const RTPPacket::shared& packet)
{
	LayerInfo info;
	VP9PayloadDescription desc;
	
	//If packet has frame markings
	if (packet->HasFrameMarkings())
	{
		//Get it from frame markings
		const auto& fm = packet->GetFrameMarks();
		//Import data
		info.temporalLayerId			= fm.temporalLayerId;
		info.spatialLayerId			= fm.layerId & 0x07;
	//Parse VP9 payload description
	} else if (desc.Parse(packet->GetMediaData(),packet->GetMaxMediaLength())) {
		//Get data
		info.temporalLayerId			= desc.temporalLayerId;
		info.spatialLayerId			= desc.spatialLayerId;
	}
	//UltraDebug("-VP9LayerSelector::GetLayerIds() | [tid:%u,sid:%u]\n",info.temporalLayerId,info.spatialLayerId);
	//Return layer info
	return info;
}