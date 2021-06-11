package resources

import (
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"strconv"
)

type IbmImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	AccountClient  *services.Account
	ProductPackageClient *services.Product_Package
}

func (imageHandler *IbmImageHandler) setterImage(imageItem datatypes.Product_Item) (irs.ImageInfo,error) {
	var imageInfo irs.ImageInfo
	defer func() {
		recover()
	}()
	if *imageItem.ItemCategory.CategoryCode == "os" {
		if *imageItem.ActiveUsagePriceCount > 0{
			if !*imageItem.CapacityRestrictedProductFlag {
				imageInfo = irs.ImageInfo{
					IId: irs.IID{NameId: *imageItem.KeyName, SystemId: strconv.Itoa(*imageItem.Id)},
					GuestOS: *imageItem.Description,
					Status: "Active",
				}
				return imageInfo, nil
			}
		}
	}
	return imageInfo, errors.New("invalid Image")
}

func (imageHandler *IbmImageHandler) setterImageTemplate(imageTemplate datatypes.Virtual_Guest_Block_Device_Template_Group) (irs.ImageInfo, error) {
	var imageInfo irs.ImageInfo
	err := errors.New("not Invalid Image")
	defer func() {
		v := recover()
		if v != nil{
			imageInfo = irs.ImageInfo{}
			err = errors.New("invalid image")
		}
	}()
	imageBlockDevices := imageTemplate.FirstChild.BlockDevices
	imageStatus := *imageTemplate.Status.Name
	for _, blockDevice := range imageBlockDevices{
		if blockDevice.DiskImage != nil {
			softwareReferences := blockDevice.DiskImage.SoftwareReferences
			if softwareReferences != nil{
				for _, softwareReference := range softwareReferences{
					if *softwareReference.SoftwareDescription.OperatingSystem == 1 {
						var guestOSName string
						if softwareReference.SoftwareDescription.LongDescription != nil {
							guestOSName = *softwareReference.SoftwareDescription.LongDescription
						}else {
							guestOSName = *softwareReference.SoftwareDescription.Name+*softwareReference.SoftwareDescription.Version
						}
						imageInfo := irs.ImageInfo{
							IId: irs.IID{
								NameId: *imageTemplate.Name,
								SystemId: *imageTemplate.GlobalIdentifier,
							},
							GuestOS: guestOSName,
							Status: imageStatus,
						}
						return imageInfo, nil
					}
				}
			}
		}
	}
	return irs.ImageInfo{}, errors.New("invalid image")
}

func (imageHandler *IbmImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageReqInfo.IId.NameId, "CreateImage()")
	// start := call.Start()
	err := errors.New(fmt.Sprintf("CreateImage Function Not Offer"))
	LoggingError(hiscallInfo, err)

	return irs.ImageInfo{}, errors.New(fmt.Sprintf("CreateImage Function Not Offer"))
}

func (imageHandler *IbmImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, "IMAGE", "ListImage()")


	productFilter := filter.Path("keyName").Eq(productName).Build()
	osItemMask :="mask[itemCategory[categoryCode],activeUsagePriceCount,capacityRestrictedProductFlag]"
	//imageTemplateMask := "mask[status,firstChild,children[blockDevices[diskImage[softwareReferences[softwareDescription,passwords]]]],globalIdentifier,name]"
	var vmImageInfos []*irs.ImageInfo

	start := call.Start()

	products, err:= imageHandler.ProductPackageClient.Filter(productFilter).GetAllObjects()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	if !(len(products) > 0){
		err = errors.New(	fmt.Sprintf("not Exist %s Package",productName))
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	// ProductItemImage
	packageSoftwareItems, err := imageHandler.ProductPackageClient.Mask(osItemMask).Id(*products[0].Id).GetActiveSoftwareItems()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	for _, item := range packageSoftwareItems{
		imageInfo, err := imageHandler.setterImage(item)
		if err == nil {
			vmImageInfos = append(vmImageInfos,&imageInfo)
		}
	}
	// ImageTemplate
	//imageTemplates , err := imageHandler.AccountClient.Mask(imageTemplateMask).GetBlockDeviceTemplateGroups()
	//for _, imageTemplate := range imageTemplates{
	//	imageInfo := imageHandler.setterImageTemplate(imageTemplate)
	//	if imageInfo != nil{
	//		vmImageInfos = append(vmImageInfos, imageInfo)
	//	}
	//}
	LoggingInfo(hiscallInfo, start)
	return vmImageInfos, nil
}

func (imageHandler *IbmImageHandler) GetImage(iid irs.IID) (irs.ImageInfo, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, iid.NameId, "GetImage()")

	productFilter := filter.Path("keyName").Eq(productName).Build()

	products, err := imageHandler.ProductPackageClient.Filter(productFilter).GetAllObjects()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{}, err
	}
	if iid.SystemId != ""{
		systemId, err := strconv.Atoi(iid.SystemId)
		if err != nil {
			// imageTemplate
			//imageTemplateMask := "mask[status,firstChild,children[blockDevices[diskImage[softwareReferences[softwareDescription,passwords]]]],globalIdentifier,name]"
			//imageTemplateFilter := filter.Path("blockDeviceTemplateGroups.globalIdentifier").Eq(iid.SystemId).Build()
			//imageTemplates , err :=  imageHandler.AccountClient.Mask(imageTemplateMask).Filter(imageTemplateFilter).GetBlockDeviceTemplateGroups()
			//if err != nil{
			//	return irs.ImageInfo{}, err
			//}
			//if len(imageTemplates) > 0 {
			//	imageInfo := imageHandler.setterImageTemplate(imageTemplates[0])
			//	if imageInfo != nil{
			//		return *imageInfo, nil
			//	}
			//}
			LoggingError(hiscallInfo, err)
			return irs.ImageInfo{},err
		} else {
			// ProductID
			itemFilter := filter.Path("activeSoftwareItems.id").Eq(systemId).Build()
			osItemMask :="mask[itemCategory[categoryCode],activeUsagePriceCount,capacityRestrictedProductFlag]"
			packageSoftwareItem, err := imageHandler.ProductPackageClient.Id(*products[0].Id).Mask(osItemMask).Filter(itemFilter).GetActiveSoftwareItems()
			if err != nil{
				LoggingError(hiscallInfo, err)
				return irs.ImageInfo{}, err
			}
			if len(packageSoftwareItem) > 0 {
				imageInfo, err  := imageHandler.setterImage(packageSoftwareItem[0])
				if err != nil{
					LoggingError(hiscallInfo, err)
					return irs.ImageInfo{}, err
				}
				return imageInfo, nil
			}
			err = errors.New(fmt.Sprintf("Not Exist %s",iid.NameId))
			LoggingError(hiscallInfo, err)
			return irs.ImageInfo{},err
		}
	}else{
		itemFilter := filter.Path("activeSoftwareItems.keyName").Eq(iid.NameId).Build()
		osItemMask :="mask[itemCategory[categoryCode],activeUsagePriceCount,capacityRestrictedProductFlag]"
		packageSoftwareItem, err := imageHandler.ProductPackageClient.Id(*products[0].Id).Mask(osItemMask).Filter(itemFilter).GetActiveSoftwareItems()
		if err != nil{
			LoggingError(hiscallInfo, err)
			return irs.ImageInfo{}, err
		}
		if len(packageSoftwareItem) > 0 {
			imageInfo, err  := imageHandler.setterImage(packageSoftwareItem[0])
			if err != nil{
				LoggingError(hiscallInfo, err)
				return irs.ImageInfo{}, err
			}
			return imageInfo, nil
		}
		err = errors.New(fmt.Sprintf("Not Exist %s",iid.NameId))
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{},err
	}

}

func (imageHandler *IbmImageHandler) DeleteImage(iid irs.IID) (bool, error){

	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, iid.NameId, "DeleteImage()")
	// start := call.Start()
	err := errors.New(fmt.Sprintf("DeleteImage Function Not Offer"))
	LoggingError(hiscallInfo, err)
	return false, err
}
