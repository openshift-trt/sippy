import './CheckboxList.css'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Checkbox,
  FormControl,
  FormControlLabel,
  FormGroup,
} from '@material-ui/core'
import { ExpandMore } from '@material-ui/icons'
import { makeStyles } from '@material-ui/core/styles'
import PropTypes from 'prop-types'
import React, { useState } from 'react'
import Typography from '@material-ui/core/Typography'

export default function CheckBoxList(props) {
  const useStyles = makeStyles((theme) => ({
    formControl: {
      margin: theme.spacing(1),
      minWidth: '20px',
    },
    selectEmpty: {
      marginTop: theme.spacing(2),
    },
    headerName: {
      width: '220px',
      padding: '0px',
      margin: '0px',
    },
    summary: {
      backgroundColor: 'rgb(0, 153, 255)',
      margin: '0px !important',
      padding: '0px',
    },
  }))

  const classes = useStyles()
  const checkedItems = props.checkedItems
  const setCheckedItems = props.setCheckedItems
  const handleChange = (event) => {
    const item = event.target.name
    const isChecked = event.target.checked
    if (isChecked) {
      setCheckedItems([...checkedItems, item])
    } else {
      setCheckedItems(
        checkedItems.filter((checkedItem) => checkedItem !== item)
      )
    }
  }

  return (
    <FormControl className={classes.formControl} component="fieldset">
      <Accordion className={classes.headerName}>
        <AccordionSummary
          className={classes.summary}
          expandIcon={<ExpandMore />}
        >
          <Typography className="checkboxlist-label">
            {props.headerName}
          </Typography>
        </AccordionSummary>
        <AccordionDetails>
          <FormGroup>
            {props.displayList.map((item) => (
              <FormControlLabel
                key={item}
                control={
                  <Checkbox
                    checked={checkedItems.includes(item)}
                    onChange={handleChange}
                    name={item}
                  />
                }
                label={item}
              />
            ))}
          </FormGroup>
        </AccordionDetails>
      </Accordion>
    </FormControl>
  )
}

CheckBoxList.propTypes = {
  headerName: PropTypes.string,
  displayList: PropTypes.array,
  checkedItems: PropTypes.array,
  setCheckedItems: PropTypes.func,
}
