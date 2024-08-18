import './ComponentReadiness.css'
import { CompReadyVarsContext } from './CompReadyVars'
import {
  dateFormat,
  formatLongDate,
  getUpdatedUrlParts,
} from './CompReadyUtils'
import { FormControl, Grid, InputLabel, MenuItem, Select } from '@mui/material'
import { Fragment } from 'react'
import { makeStyles, useTheme } from '@mui/styles'
import { useHistory } from 'react-router-dom'
import AdvancedOptions from './AdvancedOptions'
import Button from '@mui/material/Button'
import GroupByCheckboxList from './GroupByCheckboxList'
import IncludeVariantCheckBoxList from './IncludeVariantCheckboxList'
import PropTypes from 'prop-types'
import React, { useContext } from 'react'
import ReleaseSelector from './ReleaseSelector'
import Tooltip from '@mui/material/Tooltip'

export const useStyles = makeStyles((theme) => ({
  crRelease: {
    textAlign: 'center',
    marginBottom: 50,
    fontWeight: 'bold',
    padding: 5,
    backgroundColor:
      theme.palette.mode == 'dark'
        ? theme.palette.grey[800]
        : theme.palette.grey[300],
  },
}))

export default function CompReadyMainInputs(props) {
  const theme = useTheme()
  const classes = useStyles(theme)
  // checkBoxHiddenIncludeVariants defines what variants are excluded when creating Include Variant CheckBox
  // This could also be deduced from varsContext.dbGroupByVariants
  const checkBoxHiddenIncludeVariants = new Set([
    'Aggregation',
    'FromRelease',
    'FromReleaseMajor',
    'FromReleaseMinor',
    'NetworkAccess',
    'NetworkStack',
    'Release',
    'ReleaseMajor',
    'ReleaseMinor',
    'Scheduler',
    'SecurityMode',
  ])

  const varsContext = useContext(CompReadyVarsContext)
  const history = useHistory()
  const compReadyEnvOptions = (
    <div>
      <GroupByCheckboxList
        headerName="Group By"
        displayList={varsContext.dbGroupByVariants}
        checkedItems={varsContext.columnGroupByCheckedItems}
        setCheckedItems={varsContext.setColumnGroupByCheckedItems}
      />
      {Object.keys(varsContext.allJobVariants)
        .filter((key) => !checkBoxHiddenIncludeVariants.has(key))
        .map((variant) => (
          <IncludeVariantCheckBoxList
            key={variant}
            variantGroupName={variant}
          />
        ))}
      <AdvancedOptions
        headerName="Advanced"
        confidence={varsContext.confidence}
        pity={varsContext.pity}
        minFail={varsContext.minFail}
        ignoreMissing={varsContext.ignoreMissing}
        ignoreDisruption={varsContext.ignoreDisruption}
        setConfidence={varsContext.setConfidence}
        setPity={varsContext.setPity}
        setMinFail={varsContext.setMinFail}
        setIgnoreMissing={varsContext.setIgnoreMissing}
        setIgnoreDisruption={varsContext.setIgnoreDisruption}
      ></AdvancedOptions>
    </div>
  )
  return (
    <Fragment>
      <div className={classes.crRelease}>
        <FormControl variant="standard">
          <InputLabel>View</InputLabel>
          <Select
            variant="standard"
            value={varsContext.view}
            onChange={(e) => {
              console.log('changed view to: ' + e.target.value)
              varsContext.setView(e.target.value)
              history.push('/component_readiness/main?view=' + e.target.value)
              // TODO: update all query param inputs below to match the selected view
              varsContext.views.forEach(function (item) {
                if (item.name === e.target.value) {
                  varsContext.setBaseRelease(item.base_release.release)
                  varsContext.setSampleRelease(item.sample_release.release)
                }
              })
            }}
          >
            {varsContext.views.map((v, index) => (
              <MenuItem key={index} value={v.name}>
                {v.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </div>

      <div className="cr-report-button">
        <Button
          size="large"
          variant="contained"
          color="primary"
          to={'/component_readiness/main' + getUpdatedUrlParts(varsContext)}
          onClick={varsContext.handleGenerateReport}
        >
          <Tooltip
            title={
              'Click here to generate a report that compares the release you wish to evaluate\
                           against a historical (previous) release using all the specific parameters specified'
            }
          >
            <Fragment>Generate Custom Report</Fragment>
          </Tooltip>
        </Button>
      </div>

      <div className={classes.crRelease}>
        <ReleaseSelector
          label="Sample Release"
          tooltip="Release and dates to compare for regression against the basis (historical) release"
          version={varsContext.sampleRelease}
          onChange={varsContext.setSampleReleaseWithDates}
          startTime={formatLongDate(varsContext.sampleStartTime, dateFormat)}
          setStartTime={varsContext.setSampleStartTime}
          endTime={formatLongDate(varsContext.sampleEndTime, dateFormat)}
          setEndTime={varsContext.setSampleEndTime}
          pullRequestSupport={true}
          pullRequestOrg={varsContext.samplePROrg}
          setPullRequestOrg={varsContext.setSamplePROrg}
          pullRequestRepo={varsContext.samplePRRepo}
          setPullRequestRepo={varsContext.setSamplePRRepo}
          pullRequestNumber={varsContext.samplePRNumber}
          setPullRequestNumber={varsContext.setSamplePRNumber}
        ></ReleaseSelector>
      </div>
      <div className={classes.crRelease}>
        <ReleaseSelector
          version={varsContext.baseRelease}
          label="Basis Release"
          tooltip="Release and dates to specify a historical record of how tests have performed"
          onChange={varsContext.setBaseReleaseWithDates}
          startTime={formatLongDate(varsContext.baseStartTime, dateFormat)}
          setStartTime={varsContext.setBaseStartTime}
          endTime={formatLongDate(varsContext.baseEndTime, dateFormat)}
          setEndTime={varsContext.setBaseEndTime}
        ></ReleaseSelector>
      </div>
      {props.isTestDetails ? '' : compReadyEnvOptions}
    </Fragment>
  )
}

// component and environment may be null so they are not required
CompReadyMainInputs.propTypes = {
  isTestDetails: PropTypes.bool,
}
