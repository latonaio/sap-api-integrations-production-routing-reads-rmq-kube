package sap_api_caller

import (
	"fmt"
	"io/ioutil"
	"net/http"
	sap_api_output_formatter "sap-api-integrations-production-routing-reads-rmq-kube/SAP_API_Output_Formatter"
	"strings"
	"sync"

	"github.com/latonaio/golang-logging-library/logger"
	"golang.org/x/xerrors"
)

type RMQOutputter interface {
	Send(sendQueue string, payload map[string]interface{}) error
}

type SAPAPICaller struct {
	baseURL      string
	apiKey       string
	outputQueues []string
	outputter    RMQOutputter
	log          *logger.Logger
}

func NewSAPAPICaller(baseUrl string, outputQueueTo []string, outputter RMQOutputter, l *logger.Logger) *SAPAPICaller {
	return &SAPAPICaller{
		baseURL:      baseUrl,
		apiKey:       GetApiKey(),
		outputQueues: outputQueueTo,
		outputter:    outputter,
		log:          l,
	}
}

func (c *SAPAPICaller) AsyncGetProductionRouting(productionRoutingGroup, productionRouting, product, plant, billOfOperationsDesc, sequenceText, operationText string, accepter []string) {
	wg := &sync.WaitGroup{}
	wg.Add(len(accepter))
	for _, fn := range accepter {
		switch fn {
		case "Header":
			func() {
				c.Header(productionRoutingGroup, productionRouting)
				wg.Done()
			}()
		case "ProductPlant":
			func() {
				c.ProductPlant(product, plant)
				wg.Done()
			}()
		case "BillOfOperationsDesc":
			func() {
				c.BillOfOperationsDesc(billOfOperationsDesc)
				wg.Done()
			}()
		case "SequenceText":
			func() {
				c.SequenceText(sequenceText)
				wg.Done()
			}()
		case "OperationText":
			func() {
				c.OperationText(operationText)
				wg.Done()
			}()
		default:
			wg.Done()
		}
	}

	wg.Wait()
}

func (c *SAPAPICaller) Header(productionRoutingGroup, productionRouting string) {
	headerData, err := c.callProductionRoutingSrvAPIRequirementHeader("ProductionRoutingHeader", productionRoutingGroup, productionRouting)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": headerData, "function": "ProductionRoutingHeader"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(headerData)

	materialAssignmentData, err := c.callToMaterialAssignment(headerData[0].ToMaterialAssignment)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": materialAssignmentData, "function": "ProductionRoutingMaterialAssignment"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(materialAssignmentData)

	sequenceData, err := c.callToSequence(headerData[0].ToSequence)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": sequenceData, "function": "ProductionRoutingSequence"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(sequenceData)

	operationData, err := c.callToOperation(sequenceData[0].ToOperation)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": operationData, "function": "ProductionRoutingOperation"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(operationData)
	
	componentAllocationData, err := c.callToComponentAllocation(operationData[0].ToComponentAllocation)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": componentAllocationData, "function": "ProductionRoutingComponentAllocation"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(componentAllocationData)
}

func (c *SAPAPICaller) callProductionRoutingSrvAPIRequirementHeader(api, productionRoutingGroup, productionRouting string) ([]sap_api_output_formatter.Header, error) {
	url := strings.Join([]string{c.baseURL, "API_PRODUCTION_ROUTING", api}, "/")
	req, _ := http.NewRequest("GET", url, nil)

	c.setHeaderAPIKeyAccept(req)
	c.getQueryWithHeader(req, productionRoutingGroup, productionRouting)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToHeader(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) callToMaterialAssignment(url string) ([]sap_api_output_formatter.ToMaterialAssignment, error) {
	req, _ := http.NewRequest("GET", url, nil)
	c.setHeaderAPIKeyAccept(req)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToToMaterialAssignment(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) callToSequence(url string) ([]sap_api_output_formatter.ToSequence, error) {
	req, _ := http.NewRequest("GET", url, nil)
	c.setHeaderAPIKeyAccept(req)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToToSequence(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) callToOperation(url string) ([]sap_api_output_formatter.ToOperation, error) {
	req, _ := http.NewRequest("GET", url, nil)
	c.setHeaderAPIKeyAccept(req)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToToOperation(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) callToComponentAllocation(url string) ([]sap_api_output_formatter.ToComponentAllocation, error) {
	req, _ := http.NewRequest("GET", url, nil)
	c.setHeaderAPIKeyAccept(req)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToToComponentAllocation(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) ProductPlant(product, plant string) {
	data, err := c.callProductionRoutingSrvAPIRequirementProductPlant("ProductionRoutingMatlAssgmt", product, plant)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": data, "function": "ProductionRoutingMaterialAssignment"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(data)

	sequenceData, err := c.callToSequence(
		fmt.Sprintf("%s/API_PRODUCTION_ROUTING/ProductionRoutingHeader(ProductionRoutingGroup='%s',ProductionRouting='%s',ProductionRoutingInternalVers='%s')/to_Sequence",
			c.baseURL, data[0].ProductionRoutingGroup, data[0].ProductionRouting, data[0].ProductionRtgMatlAssgmtIntVers,
		),
	)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": sequenceData, "function": "ProductionRoutingSequence"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(sequenceData)

	operationData, err := c.callToOperation(sequenceData[0].ToOperation)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": operationData, "function": "ProductionRoutingOperation"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(operationData)

}

func (c *SAPAPICaller) callProductionRoutingSrvAPIRequirementProductPlant(api, product, plant string) ([]sap_api_output_formatter.MaterialAssignment, error) {
	url := strings.Join([]string{c.baseURL, "API_PRODUCTION_ROUTING", api}, "/")
	req, _ := http.NewRequest("GET", url, nil)

	c.setHeaderAPIKeyAccept(req)
	c.getQueryWithProductPlant(req, product, plant)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToMaterialAssignment(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) BillOfOperationsDesc(billOfOperationsDesc string) {
	data, err := c.callProductionRoutingSrvAPIRequirementBillOfOperationsDesc("ProductionRoutingHeader", billOfOperationsDesc)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": data, "function": "ProductionRoutingHeader"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(data)

}

func (c *SAPAPICaller) callProductionRoutingSrvAPIRequirementBillOfOperationsDesc(api, billOfOperationsDesc string) ([]sap_api_output_formatter.Header, error) {
	url := strings.Join([]string{c.baseURL, "API_PRODUCTION_ROUTING", api}, "/")
	req, _ := http.NewRequest("GET", url, nil)

	c.setHeaderAPIKeyAccept(req)
	c.getQueryWithBillOfOperationsDesc(req, billOfOperationsDesc)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToHeader(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) SequenceText(sequenceText string) {
	data, err := c.callProductionRoutingSrvAPIRequirementSequenceText("ProductionRoutingSequence", sequenceText)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": data, "function": "ProductionRoutingSequence"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(data)

}

func (c *SAPAPICaller) callProductionRoutingSrvAPIRequirementSequenceText(api, sequenceText string) ([]sap_api_output_formatter.Sequence, error) {
	url := strings.Join([]string{c.baseURL, "API_PRODUCTION_ROUTING", api}, "/")
	req, _ := http.NewRequest("GET", url, nil)

	c.setHeaderAPIKeyAccept(req)
	c.getQueryWithSequenceText(req, sequenceText)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToSequence(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) OperationText(operationText string) {
	data, err := c.callProductionRoutingSrvAPIRequirementOperationText("ProductionRoutingOperation", operationText)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": data, "function": "ProductionRoutingOperation"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(data)

}

func (c *SAPAPICaller) callProductionRoutingSrvAPIRequirementOperationText(api, operationText string) ([]sap_api_output_formatter.Operation, error) {
	url := strings.Join([]string{c.baseURL, "API_PRODUCTION_ROUTING", api}, "/")
	req, _ := http.NewRequest("GET", url, nil)

	c.setHeaderAPIKeyAccept(req)
	c.getQueryWithOperationText(req, operationText)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToOperation(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) setHeaderAPIKeyAccept(req *http.Request) {
	req.Header.Set("APIKey", c.apiKey)
	req.Header.Set("Accept", "application/json")
}

func (c *SAPAPICaller) getQueryWithHeader(req *http.Request, productionRoutingGroup, productionRouting string) {
	params := req.URL.Query()
	params.Add("$filter", fmt.Sprintf("ProductionRoutingGroup eq '%s' and ProductionRouting eq '%s'", productionRoutingGroup, productionRouting))
	req.URL.RawQuery = params.Encode()
}

func (c *SAPAPICaller) getQueryWithProductPlant(req *http.Request, product, plant string) {
	params := req.URL.Query()
	params.Add("$filter", fmt.Sprintf("Product eq '%s' and Plant eq '%s'", product, plant))
	req.URL.RawQuery = params.Encode()
}

func (c *SAPAPICaller) getQueryWithBillOfOperationsDesc(req *http.Request, billOfOperationsDesc string) {
	params := req.URL.Query()
	params.Add("$filter", fmt.Sprintf("substringof('%s', BillOfOperationsDesc)", billOfOperationsDesc))
	req.URL.RawQuery = params.Encode()
}

func (c *SAPAPICaller) getQueryWithSequenceText(req *http.Request, sequenceText string) {
	params := req.URL.Query()
	params.Add("$filter", fmt.Sprintf("substringof('%s', SequenceText)", sequenceText))
	req.URL.RawQuery = params.Encode()
}

func (c *SAPAPICaller) getQueryWithOperationText(req *http.Request, operationText string) {
	params := req.URL.Query()
	params.Add("$filter", fmt.Sprintf("substringof('%s', OperationText)", operationText))
	req.URL.RawQuery = params.Encode()
}
